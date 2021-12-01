package app

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// WriteMsgBylconn 本地localconn发包
func (l *UdpListener) WriteMsgBylconn(add *net.UDPAddr, msg []byte) {
	i := 0
	for i < 3 {
		_, err := l.LocalConn.WriteTo(msg, add)
		if err == nil {
			return
		}
		log.Errorf("write to addr=%s faild", add.String())
		i++
	}
}

// WriteMsgToSvr 手动发包给svr的p2p服务端
func (l *UdpListener) WriteMsgToSvr(msg []byte) {
	i := 0
	//dstAddr := &net.UDPAddr{IP: net.ParseIP(l.Conf.ServerHost), Port: l.Conf.ServerPort}
	dstAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", l.Conf.ServerHost, l.Conf.ServerPort))
	if err != nil {
		log.Fatalf("解析svc的地址失败,err=%s", err.Error())
	}
	for i < 3 {
		_, err := l.LocalConn.WriteToUDP(msg, dstAddr)
		if err == nil {
			return
		}
		log.Errorf("write to svr p2p server faild,err=%s,addr=%s", err.Error(), l.LocalConn.RemoteAddr().String())
		i++
	}
}

// WriteMsgToClient 给另一侧的client客户端发包
func (l *UdpListener) WriteMsgToClient(msg []byte) {
	if l.ClientServerIp.Addr == "" {
		log.Error("没有获取到另一侧的ip地址")
		return
	}
	dstAddr := parseAddr(l.ClientServerIp.Addr)
	i := 0
	for i < 3 {
		_, err := l.LocalConn.WriteToUDP(msg, dstAddr)
		if err == nil {
			return
		}
		log.Errorf("write to  p2p client faild,err=%s,addr=%s", err.Error(), l.LocalConn.RemoteAddr().String())
		i++
	}
}

func (l *UdpListener) WriteMsgToRdp(msg []byte) (int, error) {
	return l.RdpConn.Write(msg)
}

func parseAddr(addr string) *net.UDPAddr {
	t := strings.Split(addr, ":")
	port, _ := strconv.Atoi(t[1])
	return &net.UDPAddr{
		IP:   net.ParseIP(t[0]),
		Port: port,
	}
}

// 有效期30秒 改成了tcp的状态。p2p打通后，将转成tcp监听
func (l *UdpListener) checkStatus() bool {
	if !l.Status.Status || l.Status.Time.UnixNano() < (time.Now().UnixNano()-30*time.Second.Nanoseconds()) {
		return false
	}
	return true
	//return l.Proto.Proto == PROTO_TCP
}


// Forward all requests from r to w
func proxyRequest(r net.Conn, w net.Conn) {
	defer r.Close()
	defer w.Close()

	var buffer = make([]byte, 4096000)
	for {
		n, err := r.Read(buffer)
		if err != nil {
			fmt.Printf("Unable to read from input, error: %s\n", err.Error())
			break
		}

		n, err = w.Write(buffer[:n])
		if err != nil {
			fmt.Printf("Unable to write to output, error: %s\n", err.Error())
			break
		}
	}
}

// Start a proxy server listen on fromport
// this proxy will then forward all request from fromport to toport
//
// Notice: a service must has been started on toport
func proxyStart(fromport, toport int) {
	proxyaddr := fmt.Sprintf(":%d", fromport)
	proxylistener, err := net.Listen("tcp", proxyaddr)
	if err != nil {
		fmt.Println("Unable to listen on: %s, error: %s\n", proxyaddr, err.Error())
		os.Exit(1)
	}
	defer proxylistener.Close()

	for {
		proxyconn, err := proxylistener.Accept()
		if err != nil {
			fmt.Printf("Unable to accept a request, error: %s\n", err.Error())
			continue
		}

		// Read a header firstly in case you could have opportunity to check request
		// whether to decline or proceed the request
		buffer := make([]byte, 1024)
		n, err := proxyconn.Read(buffer)
		if err != nil {
			fmt.Printf("Unable to read from input, error: %s\n", err.Error())
			continue
		}

		// TODO
		// Your choice to make decision based on request header

		targetaddr := fmt.Sprintf("localhost:%d", toport);
		targetconn, err := net.Dial("tcp", targetaddr)
		if err != nil {
			fmt.Println("Unable to connect to: %s, error: %s\n", targetaddr, err.Error())
			proxyconn.Close()
			continue
		}

		n, err = targetconn.Write(buffer[:n])
		if err != nil {
			fmt.Printf("Unable to write to output, error: %s\n", err.Error())
			proxyconn.Close()
			targetconn.Close()
			continue
		}

		go proxyRequest(proxyconn, targetconn)
		go proxyRequest(targetconn, proxyconn)
	}
}
