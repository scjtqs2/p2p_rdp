package app

import (
	"encoding/json"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"net"
)

func (l *UdpListener) initRdpListener() {
	var err error
	switch l.Conf.Type {
	case common.CLIENT_CLIENT_TYPE:
		//初始化listener
		l.RdpConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: l.Conf.RdpP2pPort})
		if err != nil {
			log.Fatalf("init rdp listener faild port=%d ,err=%s", l.Conf.RdpP2pPort, err.Error())
		}
	case common.CLIENT_SERVER_TYPE:
		//初始化udp client
		l.RdpConn, err = net.DialUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: l.Conf.RdpP2pPort}, parseAddr("127.0.0.1:3389"))
		if err != nil {
			log.Fatalf("init rdp client faild err=%s", err.Error())
		}
	}
}

func (l *UdpListener) RdpHandler() {
	for {
		data := make([]byte, 1024)
		n,remoteAddr,err := l.RdpConn.ReadFromUDP(data)
		l.RdpAddr=remoteAddr.String()
		if err != nil {
			log.Errorf("error during read: %s", err.Error())
		} else {
			msg,_:=json.Marshal(&common.UDPMsg{Code: 2,Data: data[:n]})
			//转发到远程client
			l.WriteMsgToClient(msg)
		}
	}
}
