package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"time"
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
	for {
		l.RdpConn, err = l.RdpListener.Accept()
		if err != nil {
			fmt.Println("accept failed, err:", err)
			continue
		}
		go l.rdpProcessTrace(l.RdpConn)
	}
}

func (l *UdpListener) rdpProcessTrace(rdpConn net.Conn) {
	data := make([]byte, 6000)

	reader := bufio.NewReader(rdpConn)

	n, err := reader.Read(data[:])
	log.Infof("msg n=%d,err=%v,data=%s", n, err, string(data[:n]))
	if err != nil {
		log.Errorf("error during read: %s", err.Error())
	} else {
		msg, _ := json.Marshal(&common.UDPMsg{Code: 2, Data: data[:n]})
		//转发到远程client
		l.WriteMsgToClient(msg)
	}
}

func (l *UdpListener) rdpClientProcess() {
	var err error
	//初始化udp client
	l.RdpConn, err = net.DialTimeout("tcp", "127.0.0.1:3389", 5*time.Second)
	if err != nil {
		log.Fatalf("init rdp client faild err=%s", err.Error())
	}
	for {
		data := make([]byte, 6000)
		n, err := l.RdpConn.Read(data[:])
		log.Infof("n=%d,err=%v", n, err)
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("error during read: %s", err.Error())
		} else {
			msg, _ := json.Marshal(&common.UDPMsg{Code: 2, Data: data[:n]})
			//转发到远程client
			l.WriteMsgToClient(msg)
		}
	}

}
