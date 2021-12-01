package app

import (
	"fmt"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
)

// rdp和本地tcp的同步

func (l *UdpListener) initRdpListener() {
	var err error
	switch l.Conf.Type {
	case common.CLIENT_CLIENT_TYPE:
		//初始化listener
		l.RdpListener, err = net.Listen("tcp", fmt.Sprintf(":%d", l.Conf.RdpP2pPort))
		if err != nil {
			log.Fatalf("init rdp listener faild port=%d ,err=%s", l.Conf.RdpP2pPort, err.Error())
		}
		go l.rdpClientProcess()
	case common.CLIENT_SERVER_TYPE:
		go l.rdpServerProcess()
	}
}

// client侧的处理流程
func (l *UdpListener) rdpClientProcess() {
	for {
		if !l.StatusTcp || l.ClientServerIp.Addr == "" {
			continue
		}
		var err error
		l.RdpConn, err = l.LocalListenTcp.AcceptTCP()
		data := make([]byte, common.PACKAGE_SIZE)
		//var data []byte
		n, err := l.RdpConn.Read(data) // 本地读取
		if err == io.EOF {
			continue
		}
		if err != nil {
			log.Errorf("error during read: %s", err.Error())
			continue
		}
		log.Infof("client tcp read n=%d", n)
		dconn, err := net.Dial("tcp", l.ClientServerIp.Addr)
		n, err = dconn.Write(data[:n]) // 写入给远程tcp连接
		if err != nil {
			l.RdpConn.Close()
			dconn.Close()
			continue
		}
		go proxyRequest(l.RdpConn, dconn)
		go proxyRequest(dconn, l.RdpConn)
	}
}

// server侧的处理流程
func (l *UdpListener) rdpServerProcess() {
	var err error
	for {
		if !l.StatusTcp || l.ClientServerIp.Addr == "" {
			continue
		}
		//初始化udp client
		l.RdpConn, err = net.Dial("tcp", l.Conf.RemoteRdpAddr)
		if err != nil {
			log.Fatalf("init rdp client faild err=%s", err.Error())
		}
		data := make([]byte, common.PACKAGE_SIZE)
		//var data []byte
		dconn, err := net.Dial("tcp", l.ClientServerIp.Addr)
		n, err := dconn.Read(data) // 远程读取
		if err == io.EOF {
			continue
		}
		if err != nil {
			log.Errorf("error during read: %s", err.Error())
			continue
		}
		log.Infof("server tcp read n=%d", n)
		n, err = l.RdpConn.Write(data[:n]) // 写入给rdp服务
		if err != nil {
			l.RdpConn.Close()
			dconn.Close()
			continue
		}
		go proxyRequest(dconn, l.RdpConn)
		go proxyRequest(l.RdpConn, dconn)
	}
}
