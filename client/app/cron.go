package app

import (
	"encoding/json"
	"github.com/robfig/cron/v3"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

//用来发送心跳包

func (l *UdpListener) startCron() {
	l.Cron = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	//定时执行任务
	l.Cron.AddFunc("*/5 * * * * *", l.keepAliveSendToSvc)
	l.Cron.AddFunc("*/5 * * * * *", l.keepAliveSendToClient)
	l.Cron.AddFunc("*/5 * * * * *", l.makeP2P)
	l.Cron.Start()
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
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
	msg, _ := json.Marshal(req)
	l.WriteMsgToSvr(msg)
}

// keepAliveSendToClient 给client侧发送心跳包
func (l *UdpListener) keepAliveSendToClient() {
	if l.ClientServerIp.Addr != "" {
		msg, _ := json.Marshal(&common.UDPMsg{Code: common.UDP_TYPE_KEEP_ALIVE, Data: []byte("keepalive from " + l.Conf.Type)})
		l.WriteMsgToClient(msg)
	}
}

//  发送p2p的打洞数据包
func (l *UdpListener) makeP2P() {
	if l.ClientServerIp.Addr == "" {
		log.Error("没有获取到另一侧的udp地址")
		return
	}
	if l.Status {
		return
	}
	msg, _ := json.Marshal(&common.UDPMsg{
		Code: common.UDP_TYPE_BI_DIRECTION_HOLE,
		Data: []byte("我是打洞消息"),
	})
	log.Infof("开发发送打洞消息 addr=%s", l.ClientServerIp.Addr)
	l.WriteMsgToClient(msg)
}
