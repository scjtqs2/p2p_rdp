package app

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"github.com/xtaci/kcptun/generic"
	"github.com/xtaci/smux"
	"io"
	"net"
	"time"
)

const (
	ScaVengeTTL  = 600
	AutoExpire   = 60
	SmuxBuf      = 4194304
	StreamBuf    = 2097152
	KeepAlive    = 10
	MTU          = 1350
	SmuxVer      = 1
	SndWnd       = 1024
	RcvWnd       = 1024
	DSCP         = 0
	NoDelay      = 1
	Interval     = 20
	Resend       = 2
	NoCongestion = 1
	sockbuf      = 4194304
	AckNodelay   = true
	Quiet        = false
)

type timedSession struct {
	session    *smux.Session
	expiryDate time.Time
}

func (l *UdpListener) writeClientIpChange(ip string) {
	l.clientIpChange <- ip
}

// handleClient aggregates connection p1 on mux with 'writeLock'
func handleClient(session *smux.Session, p1 net.Conn, quiet bool) {
	logln := func(v ...interface{}) {
		if !quiet {
			log.Println(v...)
		}
	}
	defer p1.Close()
	p2, err := session.OpenStream()
	if err != nil {
		logln(err)
		return
	}

	defer p2.Close()

	logln("stream opened", "in:", p1.RemoteAddr(), "out:", fmt.Sprint(p2.RemoteAddr(), "(", p2.ID(), ")"))
	defer logln("stream closed", "in:", p1.RemoteAddr(), "out:", fmt.Sprint(p2.RemoteAddr(), "(", p2.ID(), ")"))

	// start tunnel & wait for tunnel termination
	streamCopy := func(dst io.Writer, src io.ReadCloser) {
		if _, err := generic.Copy(dst, src); err != nil {
			// report protocol error
			if err == smux.ErrInvalidProtocol {
				log.Println("smux", err, "in:", p1.RemoteAddr(), "out:", fmt.Sprint(p2.RemoteAddr(), "(", p2.ID(), ")"))
			}
		}
		p1.Close()
		p2.Close()
	}

	go streamCopy(p1, p2)
	streamCopy(p2, p1)
}

func scavenger(ch chan timedSession) {
	// When AutoExpire is set to 0 (default), sessionList will keep empty.
	// Then this routine won't need to do anything; thus just terminate it.
	if AutoExpire <= 0 {
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var sessionList []timedSession
	for {
		select {
		case item := <-ch:
			sessionList = append(sessionList, timedSession{
				item.session,
				item.expiryDate.Add(time.Duration(ScaVengeTTL) * time.Second)})
		case <-ticker.C:
			if len(sessionList) == 0 {
				continue
			}

			var newList []timedSession
			for k := range sessionList {
				s := sessionList[k]
				if s.session.IsClosed() {
					log.Println("scavenger: session normally closed:", s.session.LocalAddr())
				} else if time.Now().After(s.expiryDate) {
					s.session.Close()
					log.Println("scavenger: session closed due to ttl:", s.session.LocalAddr())
				} else {
					newList = append(newList, sessionList[k])
				}
			}
			sessionList = newList
		}
	}
}

// server侧初始化 kcplistenser client侧初始化 tcplistener
func (l *UdpListener) initListener(ctx context.Context) {
	var err error
	switch l.Conf.Type {
	case common.CLIENT_CLIENT_TYPE:
		//初始化listener
		localTcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", l.Conf.RdpP2pPort))
		l.RdpListener, err = net.ListenTCP("tcp", localTcpAddr)
		if err != nil {
			log.Fatalf("init rdp listener faild port=%d ,err=%s", l.Conf.RdpP2pPort, err.Error())
		}
	case common.CLIENT_SERVER_TYPE:
		l.KcpListener, err = NewKcpServerWithUDPConn(l.ClientConn)
		if err != nil {
			log.Fatalf("init kcp listener faild err=%s", err.Error())
		}
		if err := l.KcpListener.SetDSCP(DSCP); err != nil {
			log.Printf("SetDSCP:%s", err.Error())
		}
		if err := l.KcpListener.SetReadBuffer(sockbuf); err != nil {
			log.Printf("SetReadBuffer:%s", err.Error())
		}
		if err := l.KcpListener.SetWriteBuffer(sockbuf); err != nil {
			log.Printf("SetWriteBuffer:%s", err.Error())
		}
	}
}

func (l *UdpListener) localRdpClientListener(ctx context.Context) {
	switch l.Conf.Type {
	case common.CLIENT_CLIENT_TYPE:
		for {
			clientAddr := <-l.clientIpChange
			log.Infof("更新clientip地址 addr=%s", clientAddr)
			//rdpListener accept
			go l.rdpClientLoop(clientAddr)
		}
	case common.CLIENT_SERVER_TYPE:
		// server侧。不需要remote addr
		for {
			if conn, err := l.KcpListener.AcceptKCP(); err == nil {
				log.Println("remote address:", conn.RemoteAddr())
				conn.SetStreamMode(true)
				conn.SetWriteDelay(false)
				conn.SetMtu(MTU)
				conn.SetNoDelay(NoDelay, Interval, Resend, NoCongestion)
				conn.SetWindowSize(SndWnd, RcvWnd)
				conn.SetACKNoDelay(AckNodelay)
				go l.handleServerMux(generic.NewCompStream(conn))
			} else {
				log.Printf("%+v", err)
			}
		}
	}
}

func (l *UdpListener) rdpClientLoop(clientAddr string) {
	createConn := func() (*smux.Session, error) {
		clientConn, err := NewKcpConnWithUDPConn(l.ClientConn, clientAddr)
		if err != nil {
			log.Fatalf("new kcp client faild err:%s", err.Error())
			panic(err)
		}
		clientConn.SetStreamMode(true)
		clientConn.SetWriteDelay(false)
		clientConn.SetMtu(MTU)
		clientConn.SetNoDelay(NoDelay, Interval, Resend, NoCongestion)
		clientConn.SetWindowSize(SndWnd, RcvWnd)
		clientConn.SetACKNoDelay(AckNodelay)
		if err := clientConn.SetDSCP(DSCP); err != nil {
			log.Printf("SetDscp:%s", err.Error())
		}
		if err := clientConn.SetReadBuffer(sockbuf); err != nil {
			log.Printf("SetReadBuffer:%s", err.Error())
		}
		if err := clientConn.SetWriteBuffer(sockbuf); err != nil {
			log.Printf("SetWriteBuffer:%s", err.Error())
		}
		log.Printf("smux version:%s on connection:%s -> %s", SmuxVer, clientConn.LocalAddr().String(), clientConn.RemoteAddr().String())
		smuxConfig := smux.DefaultConfig()
		smuxConfig.Version = SmuxVer
		smuxConfig.MaxReceiveBuffer = SmuxBuf
		smuxConfig.MaxStreamBuffer = StreamBuf
		smuxConfig.KeepAliveInterval = time.Duration(KeepAlive) * time.Second
		if err := smux.VerifyConfig(smuxConfig); err != nil {
			log.Fatalf("%+v", err)
		}
		var session *smux.Session
		session, err = smux.Client(generic.NewCompStream(clientConn), smuxConfig)
		if err != nil {
			log.Errorf("createConn faild.err=%s", err.Error())
			return nil, errors.Wrap(err, "createConn()")
		}
		return session, nil
	}
	// wait until a connection is ready
	waitConn := func() *smux.Session {
		for {
			if session, err := createConn(); err == nil {
				log.Infof("kcp session created")
				return session
			} else {
				log.Println("re-connecting:", err)
				time.Sleep(time.Second)
			}
		}
	}
	// start scavenger
	chScavenger := make(chan timedSession, 128)
	go scavenger(chScavenger)
	// start listener
	numconn := uint16(1)
	muxes := make([]timedSession, numconn)
	rr := uint16(0)
	for {
		rdpConn, err := l.RdpListener.AcceptTCP()
		if err != nil {
			fmt.Println("accept failed, err:", err)
			continue
		}
		log.Printf("tcp conn started")
		idx := rr % numconn
		// do auto expiration && reconnection
		if muxes[idx].session == nil || muxes[idx].session.IsClosed() ||
			(AutoExpire > 0 && time.Now().After(muxes[idx].expiryDate)) {
			log.Printf("kcp conn start")
			muxes[idx].session = waitConn()
			muxes[idx].expiryDate = time.Now().Add(time.Duration(AutoExpire) * time.Second)
			if AutoExpire > 0 { // only when autoexpire set
				chScavenger <- muxes[idx]
			}
		}
		log.Printf("kcp conn started")
		go handleClient(muxes[idx].session, rdpConn, Quiet)
		rr++
	}
}

// handle multiplex-ed connection
func (l *UdpListener) handleServerMux(conn net.Conn) {
	log.Println("smux version:", SmuxVer, "on connection:", conn.LocalAddr(), `->`, conn.RemoteAddr())
	// stream multiplex
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = SmuxVer
	smuxConfig.MaxReceiveBuffer = SmuxBuf
	smuxConfig.MaxStreamBuffer = StreamBuf
	smuxConfig.KeepAliveInterval = time.Duration(KeepAlive) * time.Second

	mux, err := smux.Server(conn, smuxConfig)
	if err != nil {
		log.Println(err)
		return
	}
	defer mux.Close()

	for {
		stream, err := mux.AcceptStream()
		if err != nil {
			log.Println(err)
			return
		}

		go func(p1 *smux.Stream) {
			var p2 net.Conn
			var err error
			p2, err = net.Dial("tcp", l.Conf.RemoteRdpAddr)

			if err != nil {
				log.Println(err)
				p1.Close()
				return
			}
			handleServer(p1, p2, Quiet)
		}(stream)
	}
}

func handleServer(p1 *smux.Stream, p2 net.Conn, quiet bool) {
	logln := func(v ...interface{}) {
		if !quiet {
			log.Println(v...)
		}
	}

	defer p1.Close()
	defer p2.Close()

	logln("stream opened", "in:", fmt.Sprint(p1.RemoteAddr(), "(", p1.ID(), ")"), "out:", p2.RemoteAddr())
	defer logln("stream closed", "in:", fmt.Sprint(p1.RemoteAddr(), "(", p1.ID(), ")"), "out:", p2.RemoteAddr())

	// start tunnel & wait for tunnel termination
	streamCopy := func(dst io.Writer, src io.ReadCloser) {
		if _, err := generic.Copy(dst, src); err != nil {
			if err == smux.ErrInvalidProtocol {
				log.Println("smux", err, "in:", fmt.Sprint(p1.RemoteAddr(), "(", p1.ID(), ")"), "out:", p2.RemoteAddr())
			}
		}
		p1.Close()
		p2.Close()
	}

	go streamCopy(p2, p1)
	streamCopy(p1, p2)
}
