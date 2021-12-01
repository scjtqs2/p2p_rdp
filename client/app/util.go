package app

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"strconv"
	"strings"
	"time"
)

// WriteMsgBylconn 本地localconn发包
func (l *UdpListener) WriteMsgBylconn(add *net.UDPAddr, msg []byte) {
	UdpWriteChan <- UdpWrite{Addr: add, Data: msg}
}

// WriteMsgToSvr 手动发包给svr的p2p服务端
func (l *UdpListener) WriteMsgToSvr(msg []byte) {
	//dstAddr := &net.UDPAddr{IP: net.ParseIP(l.Conf.ServerHost), Port: l.Conf.ServerPort}
	dstAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", l.Conf.ServerHost, l.Conf.ServerPort))
	if err != nil {
		log.Fatalf("解析svc的地址失败,err=%s", err.Error())
	}
	UdpWriteChan <- UdpWrite{Addr: dstAddr, Data: msg}
}

// WriteMsgToClient 给另一侧的client客户端发包
func (l *UdpListener) WriteMsgToClient(msg []byte) {
	if l.ClientServerIp.Addr == "" {
		log.Error("没有获取到另一侧的ip地址")
		return
	}
	dstAddr := parseAddr(l.ClientServerIp.Addr)
	UdpWriteChan <- UdpWrite{Addr: dstAddr, Data: msg}
}

func (l *UdpListener) WriteMsgToRdp(msg []byte) {
	RdpWriteChan <- msg
}

func parseAddr(addr string) *net.UDPAddr {
	t := strings.Split(addr, ":")
	port, _ := strconv.Atoi(t[1])
	return &net.UDPAddr{
		IP:   net.ParseIP(t[0]),
		Port: port,
	}
}

// 有效期30秒
func (l *UdpListener) checkStatus() bool {
	if !l.Status.Status || l.Status.Time.UnixNano() < (time.Now().UnixNano()-30*time.Second.Nanoseconds()) {
		return false
	}
	return true
}

var UdpWriteChan = make(chan UdpWrite)

type UdpWrite struct {
	Addr *net.UDPAddr
	Data []byte
}

func (l *UdpListener) udpSendBackend() {
	for {
		udpWrite := <-UdpWriteChan
		l.LocalConn.WriteToUDP(udpWrite.Data, udpWrite.Addr)
	}
}

var RdpWriteChan = make(chan []byte)

func (l *UdpListener) rdpSendBackend() {
	for {
		msg := <-RdpWriteChan
		l.RdpConn.Write(msg)
	}
}
