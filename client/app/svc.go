package app

import (
	"encoding/json"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"net"
)

// p2p的svc服务（discover服务） 消息读取处理
func (l *UdpListener) serverReadHandle() {
	for {
		data := make([]byte, 1024)
		n, _, err := l.ServerConn.ReadFromUDP(data)
		if err != nil {
			log.Errorf("read udp package from svc faild,error %s", err.Error())
			l.initServerConn()
			continue
		}
		var msg common.Msg
		err = json.Unmarshal(data[:n], &msg)
		if err != nil {
			continue
		}
		switch msg.Type {
		case common.MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS:
			if msg.Res.Code != 0 {
				log.Errorf("对方客户端未就绪,%+v", msg.Res)
				continue
			}
			err := json.Unmarshal([]byte(msg.Res.Message), &l.ClientServerIp)
			if err != nil {
				log.Errorf("解码json失败 err %s", err.Error())
				continue
			}
		case common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS:
			if msg.Res.Code != 0 {
				log.Errorf("对方客户端未就绪,%+v", msg.Res)
				continue
			}
			err := json.Unmarshal([]byte(msg.Res.Message), &l.ClientServerIp)
			if err != nil {
				log.Errorf("解码json失败 err %s", err.Error())
				continue
			}
		}
	}
}

func (l *UdpListener) initServerConn() (err error) {
	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: l.Conf.ClientPort} // 注意端口必须固定，udp打洞，需要两侧
	dstAddr := &net.UDPAddr{IP: net.ParseIP(l.Conf.ServerHost), Port: l.Conf.ServerPort}
	l.ServerConn, err = net.DialUDP("udp", srcAddr, dstAddr)
	return err
}
