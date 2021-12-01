package app

import (
	"context"
	"encoding/json"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"time"
)

// 发送信息给svc
func (l *UdpListener) initMsgToSvc() {
	log.Infof("发送消息给svc，addr=%s:%d", l.Conf.ServerHost, l.Conf.ServerPort)
	req := &common.Msg{AppName: l.Conf.AppName}
	switch l.Conf.Type {
	case common.CLIENT_SERVER_TYPE:
		req.Type = common.CLIENT_SERVER_TYPE
	case common.CLIENT_CLIENT_TYPE:
		req.Type = common.CLIENT_CLIENT_TYPE
	}
	seq := makeSeq()
	req.Seq = seq
	req.AppName = l.Conf.AppName
	msg, _ := json.Marshal(req)
	go l.WriteMsgToSvr(msg, seq)
}

// 本地udp端口 消息读取处理
func (l *UdpListener) localReadHandle() {
	for {
		data := make([]byte, common.PACKAGE_SIZE)
		n, remodeAddr, err := l.LocalConn.ReadFromUDP(data[:])
		if err != nil {
			log.Errorf("read udp package from svc faild,error %s", err.Error())
			continue
		}
		var msg common.UDPMsg
		err = json.Unmarshal(data[:n], &msg)
		if err != nil {
			continue
		}
		ctx := context.WithValue(context.Background(), "seq", msg.Seq)
		log.WithContext(ctx)
		if msg.Seq != "" && msg.Code != common.UDP_TYPE_SEQ_RESPONSE {
			go l.returnSeq(msg.Seq, remodeAddr)
		}
		//if l.checkSeq(msg.Seq) {
		//	continue
		//}
		//l.setSeq(msg.Seq)
		//0:心跳 1:打洞消息 2:转发消息
		switch msg.Code {
		case common.UDP_TYPE_KEEP_ALIVE:
			log.Infof("心跳包 remoteAddr=%s msg=%s", remodeAddr, string(msg.Data))
			l.Status.Status = true
			l.Status.Time = time.Now()
			ctx.Done()
			continue
		case common.UDP_TYPE_BI_DIRECTION_HOLE:
			log.Infof("打洞消息 remoteAddr=%s msg=%s", remodeAddr, string(msg.Data))
			l.Status.Status = true
			l.Status.Time = time.Now()
			seq := makeSeq()
			message, _ := json.Marshal(&common.UDPMsg{Code: 0, Data: []byte("打洞成功"), Seq: seq})
			go l.WriteMsgToClient(ctx, message, seq)
		case common.UDP_TYPE_TRANCE:
			//用rdp的端口发送数据
			recv := msg.Data
			//需要提取udp的包进行拼包，再转发给tcp的rdp端口
			//l.rdpMakeTcpPackageSend(common.UDPMsg{
			//	Code:   msg.Code,
			//	Data:   recv,
			//	Seq:    msg.Seq,
			//	Count:  msg.Count,
			//	Offset: msg.Offset,
			//	Lenth:  msg.Lenth,
			//})
			log.Debugf("tcp trance n=%d,err=%v,data=%v", n, err, recv)
			go l.WriteMsgToRdp(ctx, recv)
		case common.UDP_TYPE_DISCOVERY:
			//处理和svr之间的通信
			var svcmsg common.Msg
			json.Unmarshal(msg.Data, &svcmsg)
			l.progressSvc(ctx, svcmsg)
		case common.UDP_TYPE_RDP:
			go l.rdpUdpWrite(ctx, msg.Data, msg.Addr)
		case common.UDP_TYPE_SEQ_RESPONSE:
			log.Debugf("seq res,seq:%s", msg.Seq)
			ctx.Done()
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
		err := json.Unmarshal([]byte(msg.Res.Message), &l.ClientServerIp)
		if err != nil {
			log.Errorf("MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS 解码json失败 err %s", err.Error())
		}
		go l.writeToP2P(ctx)
	case common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS:
		if msg.Res.Code != 0 {
			log.Errorf("对方客户端未就绪,%+v", msg.Res)
			return
		}
		err := json.Unmarshal([]byte(msg.Res.Message), &l.ClientServerIp)
		if err != nil {
			log.Errorf("MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS 解码json失败 err %s", err.Error())
		}
		go l.writeToP2P(ctx)
	}
}
