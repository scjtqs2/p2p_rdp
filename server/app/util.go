package app

import (
	"crypto/sha1"
	"fmt"
	"github.com/scjtqs2/kcp-go/v5"
	"github.com/scjtqs2/p2p_rdp/server/config"
	"golang.org/x/crypto/pbkdf2"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

// 生成seq
func makeSeq() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Seed(time.Now().UnixNano())
	return strconv.Itoa(r.Intn(1000000))
}

func NewUdpServer(config *config.ServerConfig) (*net.UDPConn, error) {
	return net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: config.Port})
}

func NewKcpServer(config *config.ServerConfig) (*kcp.Listener, error) {
	key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	return kcp.ListenWithOptions(fmt.Sprintf("0.0.0.0:%d", config.Port), block, 10, 3)
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

func parseAddr(addr string) *net.UDPAddr {
	t := strings.Split(addr, ":")
	port, _ := strconv.Atoi(t[1])
	return &net.UDPAddr{
		IP:   net.ParseIP(t[0]),
		Port: port,
	}
}