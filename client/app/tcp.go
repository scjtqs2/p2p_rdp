package app

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
)

func (l *UdpListener) initLocalTcpListen() {
	var err error
	l.LocalListenTcp, err = net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4zero, Port: l.Conf.ClientPort})
	if err != nil {
		log.Fatalf("init tcp listen error:%s", err.Error())
	}
	log.Info("init tcp listen")
	go l.localTcpAccept()
}

func (l *UdpListener) localTcpAccept() {
	for {
		sconn, err := l.LocalListenTcp.AcceptTCP()
		if err != nil {
			log.Infof("local tcp read faild err:%s", err.Error())
			continue
		}
		go l.localTcpHandle(sconn)
	}
}

func (l *UdpListener) localTcpHandle(sconn net.Conn) {
	defer sconn.Close()
	var err error
	if !l.StatusTcp || l.ClientServerIp.Addr == "" {
		return
	}
	log.Infof("local tcp handle")
	dconn, err := net.Dial("tcp", l.ClientServerIp.Addr)
	if err != nil {
		fmt.Printf("连接%v失败:%v\n", l.ClientServerIp.Addr, err)
		return
	}
	ExitChan := make(chan bool, 1)
	go func(sconn net.Conn, dconn net.Conn, Exit chan bool) {
		_, err := io.Copy(dconn, sconn)
		fmt.Printf("往%v发送数据失败:%v\n", l.ClientServerIp.Addr, err)
		ExitChan <- true
	}(sconn, dconn, ExitChan)
	go func(sconn net.Conn, dconn net.Conn, Exit chan bool) {
		_, err := io.Copy(sconn, dconn)
		fmt.Printf("从%v接收数据失败:%v\n", l.ClientServerIp.Addr, err)
		ExitChan <- true
	}(sconn, dconn, ExitChan)
	<-ExitChan
	dconn.Close()
}
