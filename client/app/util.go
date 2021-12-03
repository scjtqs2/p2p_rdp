package app

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/scjtqs2/kcp-go/v5"
	"github.com/scjtqs2/p2p_rdp/client/config"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/pbkdf2"
	"net"
	"reflect"
	"strconv"
	"strings"
)

// 发送信息给svc
func (l *UdpListener) ReportToSvc(ctx context.Context) {
	log.Debugf("上报ip地址给svc，addr=%s:%d", l.Conf.ServerHost, l.Conf.ServerPort)
	req := &common.Msg{AppName: l.Conf.AppName}
	switch l.Conf.Type {
	case common.CLIENT_SERVER_TYPE:
		req.Type = common.CLIENT_SERVER_TYPE
	case common.CLIENT_CLIENT_TYPE:
		req.Type = common.CLIENT_CLIENT_TYPE
	}
	seq := makeSeq()
	req.Seq = seq
	req.AppName = l.Conf.AppName
	msg, _ := json.Marshal(req)
	msg1, _ := json.Marshal(&common.UDPMsg{
		Code: common.UDP_TYPE_REPORT,
		Data: msg,
	})
	//l.Client2.Write(msg1)
	svcAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", l.Conf.ServerHost, l.Conf.ServerPort))
	if err != nil {
		log.Fatalf("错误的 svc 地址")
	}
	l.ClientConn.WriteToUDP(msg1, svcAddr)
}

func (l *UdpListener) getIpFromSvr(ctx context.Context) {
	log.Debugf("发送消息给svc，addr=%s:%d", l.Conf.ServerHost, l.Conf.ServerPort)
	req := &common.Msg{AppName: l.Conf.AppName}
	switch l.Conf.Type {
	case common.CLIENT_SERVER_TYPE:
		req.Type = common.CLIENT_SERVER_TYPE
	case common.CLIENT_CLIENT_TYPE:
		req.Type = common.CLIENT_CLIENT_TYPE
	}
	seq := makeSeq()
	req.Seq = seq
	req.AppName = l.Conf.AppName
	msg, _ := json.Marshal(req)
	msg1, _ := json.Marshal(&common.UDPMsg{
		Code: common.UDP_TYPE_GET_CLIENT_IP,
		Data: msg,
	})
	//l.Client1.Write(msg1)
	svcAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", l.Conf.ServerHost, l.Conf.ServerPort))
	if err != nil {
		log.Fatalf("错误的 svc 地址")
	}
	l.LocalConn.WriteToUDP(msg1, svcAddr)
}

func parseAddr(addr string) *net.UDPAddr {
	t := strings.Split(addr, ":")
	port, _ := strconv.Atoi(t[1])
	return &net.UDPAddr{
		IP:   net.ParseIP(t[0]),
		Port: port,
	}
}

func IsNul(i interface{}) bool {
	vi := reflect.ValueOf(i)
	if vi.Kind() == reflect.Ptr {
		return vi.IsNil()
	}
	return false
}

func NewKcpServer(config *config.ClientConfig) (*kcp.Listener, error) {
	key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	return kcp.ListenWithOptions(fmt.Sprintf("0.0.0.0:%d", config.ClientPortForP2PTrance), block, 10, 3)
}

// serves KCP protocol for a single packet connection.
func NewKcpServerWithUDPConn(conn net.PacketConn) (*kcp.Listener, error) {
	key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	return kcp.ServeConn(block, 10, 3, conn)
}

func NewKcpClient(addr string, localPort int) (*kcp.UDPSession, error) {
	key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	localAddr := &net.UDPAddr{IP: net.IPv4zero, Port: localPort}
	return kcp.DialWithOptions(addr, block, 10, 3, localAddr)
}

func NewKcpConnWithUDPConn(conn net.PacketConn, raddr string) (*kcp.UDPSession, error) {
	key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	return kcp.NewConn(raddr, block, 10, 3, conn)
}
