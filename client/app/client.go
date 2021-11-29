package app

import (
	"encoding/json"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

// 打洞
func (l *UdpListener) bidirectionHole() {
	if l.Status {
		return
	}
	if l.ClientServerIp.Addr == "" {
		return
	}
	log.Infof("udp打洞开始，addr= %s", l.ClientServerIp.Addr)
	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: l.Conf.ClientPort}
	dstAddr := parseAddr(l.ClientServerIp.Addr)
	conn, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		log.Error(err)
	}
	shakeMsg, _ := json.Marshal(&common.UDPMsg{Code: 1, Data: []byte("我是打洞消息")})
	// 向另一个peer发送一条udp消息(对方peer的nat设备会丢弃该消息,非法来源),用意是在自身的nat设备打开一条可进入的通道,这样对方peer就可以发过来udp消息
	go func() {
		for {
			if l.Status {
				log.Info("udp p2p on ,打洞 send breaked ")
				return
			}
			if _, err = conn.Write(shakeMsg); err != nil {
				log.Errorf("send handshake: %s", err.Error())
			}
			time.Sleep(10 * time.Second)
		}
	}()

	go func() {
		for {
			if l.Status {
				log.Info("udp p2p on ,打洞 read breaked ")
				return
			}
			data := make([]byte, 1024)
			n, _, err := conn.ReadFromUDP(data)
			if err != nil {
				log.Errorf("error during read: %s", err.Error())
			} else {
				log.Infof("打洞成功 收到数据:%s", data[:n])
				l.ClientConn = conn
				l.Status = true
				return
			}
		}
	}()
}

// 本地udp端口 消息读取处理
func (l *UdpListener) localReadHandle() {
	for {
		data := make([]byte, 1024)
		n, remodeAddr, err := l.LocalConn.ReadFromUDP(data)
		if err != nil {
			log.Errorf("read udp package from svc faild,error %s", err.Error())
			continue
		}
		var msg common.UDPMsg
		err = json.Unmarshal(data[:n], &msg)
		if err != nil {
			continue
		}
		//0:心跳 1:打洞消息 2:转发消息
		switch msg.Code {
		case 0:
			log.Infof("心跳包 msg=%s", string(msg.Data))
		case 1:
			log.Infof("打洞消息 msg=$s", string(msg.Data))
			message, _ := json.Marshal(&common.UDPMsg{Code: 0, Data: []byte("打洞成功")})
			l.WriteMsgBylconn(remodeAddr, message)
		case 2:
			//用rdp的端口发送数据
			l.WriteMsgToRdp(msg.Data)
		}
	}
}
