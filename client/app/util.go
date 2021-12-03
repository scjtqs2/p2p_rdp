package app

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// WriteMsgBylconn 本地localconn发包
func (l *UdpListener) WriteMsgBylconn(add *net.UDPAddr, msg []byte, seq string) {
	UdpWriteChan <- UdpWrite{Addr: add, Data: msg, Seq: seq}
}

// WriteMsgToSvr 手动发包给svr的p2p服务端
func (l *UdpListener) WriteMsgToSvr(msg []byte, seq string) {
	dstAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", l.Conf.ServerHost, l.Conf.ServerPort))
	if err != nil {
		log.Fatalf("解析svc的地址失败,err=%s", err.Error())
	}
	UdpWriteChan <- UdpWrite{Addr: dstAddr, Data: msg, Seq: seq}
}

// WriteMsgToClient 给另一侧的client客户端发包
func (l *UdpListener) WriteMsgToClient(ctx context.Context, msg []byte, seq string) {
	if l.ClientServerIp.Addr == "" {
		log.Error("没有获取到另一侧的ip地址")
		return
	}
	dstAddr := parseAddr(l.ClientServerIp.Addr)
	UdpWriteChan <- UdpWrite{Addr: dstAddr, Data: msg, Seq: seq}
}

func (l *UdpListener) WriteMsgToRdp(ctx context.Context, msg []byte) {
	RdpWriteChan <- msg
}

func parseAddr(addr string) *net.UDPAddr {
	t := strings.Split(addr, ":")
	port, _ := strconv.Atoi(t[1])
	return &net.UDPAddr{
		IP:   net.ParseIP(t[0]),
		Port: port,
	}
}

// 有效期30秒
func (l *UdpListener) checkStatus() bool {
	if !l.Status.Status || l.Status.Time.UnixNano() < (time.Now().UnixNano()-30*time.Second.Nanoseconds()) {
		return false
	}
	return true
}

var UdpWriteChan = make(chan UdpWrite, 10)

type UdpWrite struct {
	Addr *net.UDPAddr
	Data []byte
	Seq  string
}

// 增加回包校验
func (l *UdpListener) udpSendBackend(ctx context.Context) {
	for {
		udpWrite := <-UdpWriteChan
		if l.checkSeq(udpWrite.Seq) {
			return
		}
		_, err := l.LocalConn.WriteToUDP(udpWrite.Data, udpWrite.Addr)
		if err != nil {
			log.Errorf("send udp faild err:%s", err.Error())
		}
	}
}

var RdpWriteChan = make(chan []byte, 10)

func (l *UdpListener) rdpSendBackend(conn net.Conn, RdpWriteChan chan []byte, closeChan chan bool) {
	for {
		select {
		case <-closeChan:
			close(closeChan)
			return
		case msg, ok := <-RdpWriteChan:
			if !ok {
				return
			}
			_, err := conn.Write(msg)
			if err != nil {
				log.Errorf("send tcp faild err:%s", err.Error())
				RdpWriteChan <- msg
				return
			}
		}
	}
}

func IsNul(i interface{}) bool {
	vi := reflect.ValueOf(i)
	if vi.Kind() == reflect.Ptr {
		return vi.IsNil()
	}
	return false
}
