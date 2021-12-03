package app

import (
	"context"
	"encoding/json"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"net"
)

func (l *UdpListener) handleUdpLocal(contx context.Context, conn *net.UDPConn) {
	defer conn.Close()
	for {
		data := make([]byte, common.PACKAGE_SIZE)
		n, _, err := conn.ReadFromUDP(data)
		if err != nil {
			log.Errorf("read udp package from svc faild,error %s", err.Error())
			continue
		}
		var msg common.UDPMsg
		err = json.Unmarshal(data[:n], &msg)
		if err != nil {
			log.Error("json parase err:%s", err.Error())
			continue
		}
		ctx := context.WithValue(contx, "seq", msg.Seq)
		switch msg.Code {
		case common.UDP_TYPE_DISCOVERY: //正常拿到 clientip
			//处理和svr之间的通信
			var svcmsg common.Msg
			json.Unmarshal(msg.Data, &svcmsg)
			log.Debugf("从svc拿到数据 msg:%+v", svcmsg)
			l.progressSvc(ctx, svcmsg)
		case common.UDP_TYPE_DISCOVERY_FROCE_P2P: //强制触发p2p打洞
			var svcmsg common.Msg
			json.Unmarshal(msg.Data, &svcmsg)
			var clientIp common.Ip
			err := json.Unmarshal([]byte(svcmsg.Res.Message), &clientIp)
			if err != nil {
				log.Errorf("MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS 解码json失败 err %s", err.Error())
			}
			log.Infof("svc 强推消息 msg:%+v", svcmsg)
			l.ClientServerIp = clientIp
			l.makeP2P()
			//ip变更了 TODO
			if clientIp.Addr != l.ClientServerIp.Addr && l.Conf.Type == common.CLIENT_CLIENT_TYPE {
				l.writeClientIpChange(clientIp.Addr)
			}
		}
	}
}

func (l *UdpListener) progressSvc(ctx context.Context, msg common.Msg) {
	switch msg.Type {
	case common.MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS:
		if msg.Res.Code != 0 {
			log.Errorf("对方客户端未就绪,%+v", msg.Res)
			return
		}
		var clientIp common.Ip
		err := json.Unmarshal([]byte(msg.Res.Message), &clientIp)
		if err != nil {
			log.Errorf("MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS 解码json失败 err %s", err.Error())
		}
		//ip变更了 TODO
		if clientIp.Addr != l.ClientServerIp.Addr {
			l.ClientServerIp = clientIp
			log.Infof("开始kcp 发送p2p包给 client侧")
			l.makeP2P()
			if l.Conf.Type == common.CLIENT_CLIENT_TYPE {
				l.writeClientIpChange(clientIp.Addr)
			}
		} else {
			l.ClientServerIp = clientIp
		}
	case common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS:
		if msg.Res.Code != 0 {
			log.Errorf("对方客户端未就绪,%+v", msg.Res)
			return
		}
		var clientIp common.Ip
		err := json.Unmarshal([]byte(msg.Res.Message), &clientIp)
		if err != nil {
			log.Errorf("MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS 解码json失败 err %s", err.Error())
		}
		//ip变更了 TODO
		if clientIp.Addr != l.ClientServerIp.Addr {
			l.ClientServerIp = clientIp
			log.Infof("开始kcp 发送p2p包给 server侧")
			l.makeP2P()
			if l.Conf.Type == common.CLIENT_CLIENT_TYPE {
				l.writeClientIpChange(clientIp.Addr)
			}
		} else {
			l.ClientServerIp = clientIp
		}
	}
}

//  发送p2p的打洞数据包
func (l *UdpListener) makeP2P() {
	l.p2pChan <- true
}

func (l *UdpListener) sendP2PBackend(conn *net.UDPConn) {
	for {
		select {
		case <-l.p2pChan:
			if l.ClientServerIp.Addr == "" {
				log.Error("没有获取到另一侧的udp地址")
				return
			}
			seq := makeSeq()
			ctx := context.WithValue(context.TODO(), "seq", seq)
			log.WithContext(ctx)
			msg, _ := json.Marshal(&common.UDPMsg{
				Code: common.UDP_TYPE_BI_DIRECTION_HOLE,
				Data: []byte("我是打洞消息"),
				Seq:  seq,
			})
			log.Infof("开始发送打洞消息 addr=%s", l.ClientServerIp.Addr)
			dstAddr, _ := net.ResolveUDPAddr("udp", l.ClientServerIp.Addr)
			conn.WriteToUDP(msg, dstAddr)
			conn.WriteToUDP(msg, dstAddr)
		}
	}
}
