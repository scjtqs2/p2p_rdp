package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
)

func (l *UdpListener) initRdpListener() {
	var err error
	switch l.Conf.Type {
	case common.CLIENT_CLIENT_TYPE:
		//初始化listener
		l.RdpListener, err = net.Listen("tcp", fmt.Sprintf(":%d", l.Conf.RdpP2pPort))
		if err != nil {
			log.Fatalf("init rdp listener faild port=%d ,err=%s", l.Conf.RdpP2pPort, err.Error())
		}
		go l.RdpHandler()
	case common.CLIENT_SERVER_TYPE:
		go l.rdpClientProcess()
	}
}

func (l *UdpListener) RdpHandler() {
	var err error
	defer l.RdpListener.Close()
	for {
		l.RdpConn, err = l.RdpListener.Accept()
		if err != nil {
			fmt.Println("accept failed, err:", err)
			continue
		}
		var closeChan = make(chan bool, 10)
		go l.rdpSendBackend(l.RdpConn, RdpWriteChan, closeChan)
		go l.rdpProcessTrace(l.RdpConn, closeChan)
	}
}

func (l *UdpListener) rdpProcessTrace(conn net.Conn, closeChan chan bool) {
	data := make([]byte, common.PACKAGE_SIZE)
	defer conn.Close()
	defer func() {
		closeChan <- true
	}()
	//var data []byte
	reader := bufio.NewReader(conn)
	for {
		n, err := reader.Read(data)
		tcpPackage := data[:n]
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Errorf("error during read: %s", err.Error())
			return
		} else {
			seq := makeSeq()
			ctx := context.WithValue(context.TODO(), "seq", seq)
			log.WithContext(ctx)
			msg, _ := json.Marshal(&common.UDPMsg{Code: common.UDP_TYPE_TRANCE, Data: tcpPackage, Seq: seq})
			//转发到远程client
			go l.WriteMsgToClient(ctx, msg, seq)
		}
	}

}

func (l *UdpListener) rdpClientProcess() {
	var err error
	for {
		//初始化udp client
		l.RdpConn, err = net.Dial("tcp", l.Conf.RemoteRdpAddr)
		if err != nil {
			log.Fatalf("init rdp client faild err=%s", err.Error())
		}
		var closeChan = make(chan bool, 10)
		go l.rdpSendBackend(l.RdpConn, RdpWriteChan, closeChan)
		l.rdpClientReadProcess()
		closeChan <- true
		l.RdpConn.Close()
	}
}

func (l *UdpListener) rdpClientReadProcess() {
	for {
		data := make([]byte, common.PACKAGE_SIZE)
		//var data []byte
		n, err := l.RdpConn.Read(data[:])
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Errorf("error during read: %s", err.Error())
			return
		} else {
			recv := data[:n]
			seq := makeSeq()
			ctx := context.WithValue(context.TODO(), "seq", seq)
			log.WithContext(ctx)
			msg, _ := json.Marshal(&common.UDPMsg{Code: common.UDP_TYPE_TRANCE, Data: recv, Seq: seq})
			//转发到远程client
			log.Printf("rdp client read n=%d", n)
			go l.WriteMsgToClient(ctx, msg, seq)
		}
	}
}

func (l *UdpListener) initRdpUdpListen() {
	var err error
	var rdpUdpConn *net.UDPConn
	switch l.Conf.Type {
	case common.CLIENT_CLIENT_TYPE:
		//初始化listener
		rdpUdpConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: l.Conf.RdpP2pPort})
		if err != nil {
			log.Fatalf("init rdp listener faild port=%d ,err=%s", l.Conf.RdpP2pPort, err.Error())
		}
		go l.rdpUdpHandler(rdpUdpConn)
		//写数据
		go func(socket *net.UDPConn, RdpUdpWriteChan chan UdpWrite) {
			for {
				msg := <-RdpUdpWriteChan
				socket.WriteToUDP(msg.Data, msg.Addr)
			}
		}(rdpUdpConn, RdpUdpWriteChan)
	case common.CLIENT_SERVER_TYPE:
		go l.rdpUdpDiaHandler()
	}
}

var rdpClientAddr *net.UDPAddr

// client侧读取rdp数据，发送给server侧
func (l *UdpListener) rdpUdpHandler(conn *net.UDPConn) {
	defer conn.Close()
	for {
		data := make([]byte, common.PACKAGE_SIZE)
		n, addr, err := conn.ReadFromUDP(data[:])
		rdpClientAddr = addr
		if err != nil {
			log.Errorf("read udp package from rdp faild,addr=%s ,error %s", addr, err.Error())
			continue
		}
		recv := data[:n]
		seq := makeSeq()
		ctx := context.WithValue(context.TODO(), "seq", seq)
		log.WithContext(ctx)
		msg, _ := json.Marshal(&common.UDPMsg{Code: common.UDP_TYPE_RDP, Data: recv, Seq: seq})
		//转发到远程client
		go l.WriteMsgToClient(ctx, msg, seq)
	}
}

var RdpUdpWriteChan = make(chan UdpWrite, 10)

// 3389的udp直通 写入
func (l *UdpListener) rdpUdpWrite(ctx context.Context, data []byte, addr *net.UDPAddr) {
	defer ctx.Done()
	m := UdpWrite{
		Addr: addr,
		Data: data,
	}
	RdpUdpWriteChan <- m
}

// server侧的 rdp udp 读取 转发到client侧
func (l *UdpListener) rdpUdpDiaHandler() {
	for {
		dst, _ := net.ResolveUDPAddr("udp", l.Conf.RemoteRdpAddr)
		socket, err := net.DialUDP("udp", nil, dst)
		if err != nil {
			log.Errorf("连接 rdp udp失败 err:%s", err.Error())
		}
		//写数据
		var closeChan = make(chan bool, 10)
		go func(socket *net.UDPConn, RdpUdpWriteChan chan UdpWrite, closeChan chan bool) {
			for {
				select {
				case <-closeChan:
					close(closeChan)
					return
				case msg := <-RdpUdpWriteChan:
					_, err := socket.Write(msg.Data)
					if err != nil {
						log.Errorf("tcp write faild err : %s", err.Error())
						RdpUdpWriteChan <- msg
					}
				}
			}
		}(socket, RdpUdpWriteChan, closeChan)
		//度数据
		for {
			data := make([]byte, common.PACKAGE_SIZE)
			n, addr, err := socket.ReadFromUDP(data[:])
			if err != nil {
				log.Errorf("read udp package from rdp server faild,error %s", err.Error())
				break
			}
			recv := data[:n]
			seq := makeSeq()
			ctx := context.WithValue(context.TODO(), "seq", seq)
			log.WithContext(ctx)
			msg, _ := json.Marshal(&common.UDPMsg{Code: common.UDP_TYPE_RDP, Data: recv, Addr: addr, Seq: seq})
			//转发到远程client
			go l.WriteMsgToClient(ctx, msg, seq)
		}
		closeChan <- true
		socket.Close()
	}

}
