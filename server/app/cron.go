package app

// 定时清理 peers中"掉线"的记录

import (
	"github.com/robfig/cron/v3"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (l *UdpListener) startCron() {
	l.Cron = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	//定时执行任务
	l.Cron.AddFunc("* * * * *", l.clearPeers)
	l.Cron.Start()
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
}

func (l *UdpListener) clearPeers() {
	//拉取keys
	keys := l.PeersKeys()
	go func() {
		for _, v := range keys {
			appName := v
			peers := l.PeersGet(appName)
			//校验server侧 并清理
			if !checkExpire(peers.Server) {
				peers.Server.Addr = ""
				l.PeersSet(appName, peers)
			}
			//校验client 并清理
			go l.clearClientPeers(appName)
		}
	}()
}

// 用来清理clients
func (l *UdpListener) clearClientPeers(appName string) {
	peers := l.PeersGet(appName)
	var clients []Ip
	for _, v := range peers.clients {
		client := v
		if checkExpire(client) {
			clients = append(clients, client)
		}
	}
	peers = l.PeersGet(appName)
	peers.clients = clients
	l.PeersSet(appName, peers)
}

// 校验ip地址是否过期
func checkExpire(ip Ip) bool {
	check := time.Now().UnixNano() - 5*time.Minute.Nanoseconds()
	if ip.Time.UnixNano() < check {
		return false
	}
	return true
}