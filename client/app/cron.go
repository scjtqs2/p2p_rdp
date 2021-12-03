package app

import (
	"context"
	"encoding/json"
	"github.com/robfig/cron/v3"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"time"
)

//用来发送心跳包

func (l *UdpListener) startCron(ctx context.Context) {
	l.Cron = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	//定时执行任务
	l.Cron.AddFunc("*/30 * * * * *", l.keepAliveSendToSvc)
	l.Cron.AddFunc("*/30 * * * * *", l.keepAliveSendToClient)
	//l.Cron.AddFunc("*/5 * * * * *", l.makeP2P)
	l.Cron.AddFunc("*/5 * * * * *", l.checkStatusCron)
	l.Cron.Start()
}

// keepAliveSendToSvc 给svc侧发送心跳包
func (l *UdpListener) keepAliveSendToSvc() {
	req := &common.Msg{AppName: l.Conf.AppName}
	switch l.Conf.Type {
	case common.CLIENT_SERVER_TYPE:
		req.Type = common.CLIENT_SERVER_TYPE
	case common.CLIENT_CLIENT_TYPE:
		req.Type = common.CLIENT_CLIENT_TYPE
	}
	req.AppName = l.Conf.AppName
	seq := makeSeq()
	req.Seq = seq
	msg, _ := json.Marshal(req)
	go l.WriteMsgToSvr(msg, seq)
}

// keepAliveSendToClient 给client侧发送心跳包
func (l *UdpListener) keepAliveSendToClient() {
	if l.ClientServerIp.Addr != "" {
		seq := makeSeq()
		ctx := context.WithValue(context.TODO(), "seq", seq)
		log.WithContext(ctx)
		msg, _ := json.Marshal(&common.UDPMsg{Code: common.UDP_TYPE_KEEP_ALIVE, Data: []byte("keepalive from " + l.Conf.Type), Seq: seq})
		go l.WriteMsgToClient(ctx, msg, seq)
	}
}

var p2pChan = make(chan bool, 5)

//触发 p2p打洞
func (l *UdpListener) writeToP2P(ctx context.Context) {
	p2pChan <- true
}

//后台处理p2p打洞的消费
func (l *UdpListener) makeP2Pbackend() {
	for {
		<-p2pChan
		l.makeP2P()
	}
}

//  发送p2p的打洞数据包
func (l *UdpListener) makeP2P() {
	if l.ClientServerIp.Addr == "" {
		log.Error("没有获取到另一侧的udp地址")
		return
	}
	if l.checkStatus() {
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
	go l.WriteMsgToClient(ctx, msg, seq)
}

// 检测 p2p状态
func (l *UdpListener) checkStatusCron() {
	if l.Status.Time.UnixNano() < (time.Now().UnixNano() - 30*time.Second.Nanoseconds()) {
		l.Status.Status = false
	}
	if l.ClientServerIp.Addr != "" && l.ClientServerIp.Time.UnixNano() < (time.Now().UnixNano()-30*time.Second.Nanoseconds()) {
		l.ClientServerIp.Addr = ""
	}
}
