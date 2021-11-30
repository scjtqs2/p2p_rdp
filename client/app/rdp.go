package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"net"
)

func (l *UdpListener) initRdpListener() {
	var err error
	switch l.Conf.Type {
	case common.CLIENT_CLIENT_TYPE:
		//初始化listener
		l.RdpListener, err = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", l.Conf.RdpP2pPort))
		if err != nil {
			log.Fatalf("init rdp listener faild port=%d ,err=%s", l.Conf.RdpP2pPort, err.Error())
		}
		l.RdpConn, err = l.RdpListener.Accept()
	case common.CLIENT_SERVER_TYPE:
		//初始化udp client
		l.RdpConn, err = net.Dial("tcp", "127.0.0.1:3389")
		if err != nil {
			log.Fatalf("init rdp client faild err=%s", err.Error())
		}
	}
}

func (l *UdpListener) RdpHandler() {
	for {
		data := make([]byte, 1024)
		reader := bufio.NewReader(l.RdpConn)

		n, err := reader.Read(data[:])
		if err != nil {
			log.Errorf("error during read: %s", err.Error())
		} else {
			msg, _ := json.Marshal(&common.UDPMsg{Code: 2, Data: data[:n]})
			//转发到远程client
			l.WriteMsgToClient(msg)
		}
	}
}
