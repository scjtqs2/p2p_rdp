package app

import (
	"encoding/json"
	"github.com/robfig/cron/v3"
	"github.com/scjtqs2/p2p_rdp/common"
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
	l.Cron.AddFunc("*/5 * * * * *", l.keepAliveSend)
	l.Cron.Start()
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
}

func (l *UdpListener) keepAliveSend() {
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
}
