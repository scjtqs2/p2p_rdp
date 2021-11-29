package app

import (
	"github.com/robfig/cron/v3"
	"github.com/scjtqs2/p2p_rdp/client/config"
	"github.com/scjtqs2/p2p_rdp/common"
	"net"
	"time"
)

type UdpListener struct {
	Conf           *config.ClientConfig //本地配置信息
	LocalConn      *net.UDPConn         //本地监听conn
	ServerConn     *net.UDPConn         //p2p总服务端的连conn
	ClientConn     *net.UDPConn         //和server侧的客户端连接情况
	ClientServerIp common.Ip            //server侧的客户端的地址
	Status         bool                 //server侧的连接情况
	RdpConn        *net.UDPConn         //rdp的3389端口转发
	RdpAddr        string               //rdp客户端地址
	Cron           *cron.Cron
}

func (l *UdpListener) Run(config *config.ClientConfig) (err error) {
	l.Conf = config
	//固定本地端口的监听
	l.LocalConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: config.ClientPort})
	//连接总svr侧的p2p服务
	err = l.initServerConn()
	//处理svr侧的回包
	go l.serverReadHandle()
	//发送第一次打洞请求包
	go func() {
		for {
			time.Sleep(1 * time.Second)
			l.bidirectionHole()
		}
	}()
	//处理打洞请求的回包
	go l.localReadHandle()
	//初始化rdp的本地监听
	l.initRdpListener()
	//监听rdp
	go l.RdpHandler()
	//发送心跳包维活
	l.startCron()
	return nil
}
