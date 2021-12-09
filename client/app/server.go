package app

import (
	"context"
	"github.com/bluele/gcache"
	"github.com/robfig/cron/v3"
	"github.com/scjtqs2/kcp-go/v5"
	"github.com/scjtqs2/p2p_rdp/client/config"
	"github.com/scjtqs2/p2p_rdp/common"
	"net"
	"time"
)

type UdpListener struct {
	Conf           *config.ClientConfig //本地配置信息
	LocalConn      *net.UDPConn         //本地监听conn
	ClientConn     *net.UDPConn         //本地监听conn
	ClientServerIp common.Ip            //server侧的客户端的地址
	KcpListener    *kcp.Listener        //server侧的kcp listener
	RdpListener    *net.TCPListener     //rdp的3389端口转发
	RdpAddr        string               //rdp客户端地址
	Cron           *cron.Cron
	Cache          gcache.Cache
	Client2        *kcp.UDPSession // 通过 ClientConn连接 p2p
	clientIpChange chan string
	p2pChan        chan bool
}

type Status struct {
	Status bool
	Time   time.Time
}

func (l *UdpListener) Run(ctx context.Context, config *config.ClientConfig) (err error) {
	l.Conf = config
	l.ClientServerIp = common.Ip{Addr: ""}
	l.Cache = gcache.New(200).LRU().Expiration(10 * time.Second).Build()
	l.clientIpChange = make(chan string, 10)
	l.p2pChan = make(chan bool, 128)
	//固定本地端口的监听
	l.LocalConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: config.ClientPortFroSvc})
	if err != nil {
		panic(err)
	}
	l.ClientConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: config.ClientPortForP2PTrance})
	if err != nil {
		panic(err)
	}
	l.initListener(ctx)
	// 监听svc发送过来的数据
	go l.handleUdpLocal(ctx, l.LocalConn)
	l.ReportToSvc(ctx)
	l.getIpFromSvr(ctx)

	go l.localRdpClientListener(ctx)
	go l.sendP2PBackend(l.ClientConn)

	l.startCron(ctx)
	return nil
}
