package app

import (
	log "github.com/sirupsen/logrus"
	"net"
	"strconv"
	"strings"
)

// WriteMsgBylconn 本地localconn发包
func (l *UdpListener) WriteMsgBylconn(add *net.UDPAddr, msg []byte) {
	i := 0
	for i < 3 {
		_, err := l.LocalConn.WriteTo(msg, add)
		if err == nil {
			return
		}
		log.Errorf("write to addr=%s faild", add.String())
		i++
	}
}

// WriteMsgToSvr 手动发包给svr的p2p服务端
func (l *UdpListener) WriteMsgToSvr(msg []byte) {
	i := 0
	for i < 3 {
		_, err := l.ServerConn.Write(msg)
		if err == nil {
			return
		}
		log.Errorf("write to svr p2p server faild,err=%s,addr=%s", err.Error(), l.ClientConn.RemoteAddr().String())
		i++
	}
}

// WriteMsgToClient 给另一侧的client客户端发包
func (l *UdpListener) WriteMsgToClient(msg []byte) {
	if !l.Status {
		log.Error("和另一侧的p2p客户端未建立连接")
		return
	}
	i := 0
	for i < 3 {
		_, err := l.ClientConn.Write(msg)
		if err == nil {
			return
		}
		log.Errorf("write to  p2p client faild,err=%s,addr=%s", err.Error(), l.ClientConn.RemoteAddr().String())
		i++
	}
}

func (l *UdpListener) WriteMsgToRdp(msg []byte) (int, error) {
	return l.RdpConn.Write(msg)
}

func parseAddr(addr string) *net.UDPAddr {
	t := strings.Split(addr, ":")
	port, _ := strconv.Atoi(t[1])
	return &net.UDPAddr{
		IP:   net.ParseIP(t[0]),
		Port: port,
	}
}

