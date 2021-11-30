package app

import (
	"encoding/json"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
)

// 打洞
func (l *UdpListener) bidirectionHole() {
	if l.Status {
		return
	}
	log.Infof("udp打洞开始，addr= %s", l.ClientServerIp.Addr)
	shakeMsg, _ := json.Marshal(&common.UDPMsg{Code: 1, Data: []byte("我是打洞消息")})
	// 向另一个peer发送一条udp消息(对方peer的nat设备会丢弃该消息,非法来源),用意是在自身的nat设备打开一条可进入的通道,这样对方peer就可以发过来udp消息
	go l.WriteMsgToClient(shakeMsg)
}

// 发送信息给svc
func (l *UdpListener) initMsgToSvc()  {
	log.Infof("发送消息给svc，addr=%s:%d",l.Conf.ServerHost,l.Conf.ServerPort)
	req := &common.Req{}
	switch l.Conf.Type {
	case common.CLIENT_SERVER_TYPE:
		req.Type = common.CLIENT_SERVER_TYPE
	case common.CLIENT_CLIENT_TYPE:
		req.Type = common.CLIENT_CLIENT_TYPE
	}
	req.AppName = l.Conf.AppName
	msg, _ := json.Marshal(&common.Msg{
		Type:    common.MESSAGE_TYPE_KEEP_ALIVE,
		AppName: l.Conf.AppName,
	})
	req.Message = string(msg)
	l.WriteMsgToSvr(msg)
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
		case common.UDP_TYPE_KEEP_ALIVE:
			log.Infof("心跳包 msg=%s", string(msg.Data))
		case common.UDP_TYPE_BI_DIRECTION_HOLE:
			log.Infof("打洞消息 msg=$s", string(msg.Data))
			l.Status = true
			message, _ := json.Marshal(&common.UDPMsg{Code: 0, Data: []byte("打洞成功")})
			l.WriteMsgBylconn(remodeAddr, message)
		case common.UDP_TYPE_TRANCE:
			//用rdp的端口发送数据
			l.WriteMsgToRdp(msg.Data)
		case common.UDP_TYPE_DISCOVERY:
			//处理和svr之间的通信
			var svcmsg common.Msg
			json.Unmarshal(msg.Data, &svcmsg)
			l.progressSvc(svcmsg)
		}
	}
}

func (l *UdpListener) progressSvc(msg common.Msg) {
	switch msg.Type {
	case common.MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS:
		if msg.Res.Code != 0 {
			log.Errorf("对方客户端未就绪,%+v", msg.Res)
			return
		}
		err := json.Unmarshal([]byte(msg.Res.Message), &l.ClientServerIp)
		if err != nil {
			log.Errorf("解码json失败 err %s", err.Error())
		}
		l.bidirectionHole()
	case common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS:
		if msg.Res.Code != 0 {
			log.Errorf("对方客户端未就绪,%+v", msg.Res)
			return
		}
		err := json.Unmarshal([]byte(msg.Res.Message), &l.ClientServerIp)
		if err != nil {
			log.Errorf("解码json失败 err %s", err.Error())
		}
		l.bidirectionHole()
	}
}
