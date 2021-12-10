package app

import (
	"context"
	"github.com/robfig/cron/v3"
	"time"
)

//用来发送心跳包

func (l *UdpListener) startCron(ctx context.Context) {
	l.Cron = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	//定时执行任务
	l.Cron.AddFunc("*/30 * * * * *", l.keepAliveSendToSvcReport)
	l.Cron.AddFunc("*/5 * * * * *", l.keepaliveSendToSvcGetIp)
	l.Cron.AddFunc("*/5 * * * * *", l.checkStatusCron)
	l.Cron.Start()
}

// keepAliveSendToSvc 给svc侧发送心跳包
func (l *UdpListener) keepAliveSendToSvcReport() {
	l.ReportToSvc(context.Background())
}

func (l *UdpListener) keepaliveSendToSvcGetIp() {
	l.getIpFromSvr(context.Background())
}

// 检测 p2p状态
func (l *UdpListener) checkStatusCron() {
	if l.ClientServerIp.Addr == "" {
		return
	}
	if l.ClientServerIp.Addr != "" && l.ClientServerIp.Time.UnixNano() < (time.Now().UnixNano()-30*time.Second.Nanoseconds()) {
		l.ClientServerIp.Addr = ""
	}
}
