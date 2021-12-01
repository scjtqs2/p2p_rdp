package app

import (
	"github.com/robfig/cron/v3"
	"github.com/scjtqs2/p2p_rdp/client/config"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

type UdpListener struct {
	Conf           *config.ClientConfig //本地配置信息
	LocalConn      *net.UDPConn         //本地监听conn
	LocalListenTcp *net.TCPListener     //改成tcp后的listen
	ClientServerIp common.Ip            //server侧的客户端的地址
	Status         *Status              //server侧的连接情况
	RdpConn        net.Conn             //rdp的3389端口转发
	RdpListener    net.Listener         //rdp的3389端口转发
	RdpAddr        string               //rdp客户端地址
	Cron           *cron.Cron
	Proto          listenStatus //监听的协议
	StatusTcp      bool
}

type Status struct {
	Status bool
	Time   time.Time
}

func (l *UdpListener) Run(config *config.ClientConfig) (err error) {
	l.Conf = config
	l.Status = &Status{
		Status: false,
	}
	//固定本地端口的监听
	l.LocalConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: config.ClientPort})
	if err != nil {
		log.Fatalf("init udp listen err:%s", err.Error())
	}
	//l.LocalConn, err = net.DialUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: config.ClientPort},&net.UDPAddr{IP: net.ParseIP(l.Conf.ServerHost), Port: l.Conf.ServerPort})
	go l.localReadHandle()
	l.initLocalTcpListen()
	//发送初始消息到svc
	l.initMsgToSvc()

	//初始化rdp的本地监听
	l.initRdpListener()

	//发送心跳包维活
	l.startCron()
	return nil
}