package app

import (
	"errors"
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
	ClientServerIp Ip                   //server侧的客户端的地址
	Status         bool                 //server侧的连接情况
	RdpConn        *net.UDPConn         //rdp的3389端口转发
}

type Ip struct {
	Addr string
	Time time.Time
}

func (l *UdpListener) Run(config *config.ClientConfig) (err error) {
	//固定本地端口的监听
	l.LocalConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: config.ClientPort})
	//连接总svr侧的p2p服务
	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: config.ClientPort} // 注意端口必须固定，udp打洞，需要两侧
	dstAddr := &net.UDPAddr{IP: net.ParseIP(config.ServerHost), Port: config.ServerPort}
	l.ServerConn, err = net.DialUDP("udp", srcAddr, dstAddr)
	//发送第一次打洞请求包
	//处理打洞请求的回包
	//监听local端口收到的总svr侧p2p服务端推过来的信息
	//发送心跳包维活
	switch config.Type {
	case common.CLIENT_CLIENT_TYPE:
	case common.CLIENT_SERVER_TYPE:
	default:
		return errors.New("错误的配置信息")
	}
	return nil
}
