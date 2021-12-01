package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/bluele/gcache"
	"github.com/scjtqs2/p2p_rdp/common"
	log "github.com/sirupsen/logrus"
	"io"
	"math"
	"math/rand"
	"net"
	"strconv"
	"time"
)

func (l *UdpListener) initRdpListener() {
	var err error
	switch l.Conf.Type {
	case common.CLIENT_CLIENT_TYPE:
		//初始化listener
		l.RdpListener, err = net.Listen("tcp", fmt.Sprintf(":%d", l.Conf.RdpP2pPort))
		if err != nil {
			log.Fatalf("init rdp listener faild port=%d ,err=%s", l.Conf.RdpP2pPort, err.Error())
		}
		go l.RdpHandler()
	case common.CLIENT_SERVER_TYPE:
		go l.rdpClientProcess()
	}
	// 添加后台拼包
	go l.rdpMakeTcpPackageSendBackend()
}

func (l *UdpListener) RdpHandler() {
	var err error
	for {
		l.RdpConn, err = l.RdpListener.Accept()
		if err != nil {
			fmt.Println("accept failed, err:", err)
			continue
		}
		go l.rdpProcessTrace(l.RdpConn)
	}
}

func (l *UdpListener) rdpProcessTrace(rdpConn net.Conn) {
	//data := make([]byte, 6000)
	var data []byte
	reader := bufio.NewReader(rdpConn)

	n, err := reader.Read(data[:])
	tcpPackage := data[:n]
	log.Infof("n=%d , err=%v,data=%v", n, err, tcpPackage)
	if err != nil {
		log.Errorf("error during read: %s", err.Error())
	} else {
		//msg, _ := json.Marshal(&common.UDPMsg{Code: 2, Data: data[:n]})
		//转发到远程client
		//l.WriteMsgToClient(msg)
		l.rdpMakeUdpPackageSend(tcpPackage)
	}
}

func (l *UdpListener) rdpClientProcess() {
	var err error
	//初始化udp client
	l.RdpConn, err = net.DialTimeout("tcp", l.Conf.RemoteRdpAddr, 5*time.Second)
	if err != nil {
		log.Fatalf("init rdp client faild err=%s", err.Error())
	}
	for {
		//data := make([]byte, 6000)
		var data []byte
		n, err := l.RdpConn.Read(data[:])
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("error during read: %s", err.Error())
		} else {
			//msg, _ := json.Marshal(&common.UDPMsg{Code: 2, Data: data[:n]})
			//转发到远程client
			//l.WriteMsgToClient(msg)
			l.rdpMakeUdpPackageSend(data[:n])
		}
	}
}

//rdpMakeUdpPackageSend 将tcp的包拆分成多个udp的包
func (l *UdpListener) rdpMakeUdpPackageSend(data []byte) {
	rand.Seed(time.Now().UnixNano())
	seq := rand.Intn(100000)
	packageLen := len(data)
	page := int(math.Ceil(float64(packageLen) / float64(common.PACKAGE_SIZE)))
	for i := 1; i <= page; i++ {
		sliceStart := (i - 1) * common.PACKAGE_SIZE
		sliceEnd := sliceStart + common.PACKAGE_SIZE
		if sliceEnd > packageLen {
			sliceEnd = packageLen
		}
		msg, _ := json.Marshal(&common.UDPMsg{Code: common.UDP_TYPE_TRANCE, Data: data[sliceStart:sliceEnd], Seq: seq, Offset: i, Count: page, Lenth: packageLen})
		log.Infof("tcp 写包 msg=%+V", msg)
		l.WriteMsgToClient(msg)
	}
}

var cache = gcache.New(20).LRU().Build()

const cache_key = "CACHE_FOR_TOKEN_"
const cache_key_package = "CACHE_FOR_PACKAGE_"

var packageChan = make(chan common.UDPMsg, 10)

// rdpMakeTcpPackageSend  将拆分了的udp包拼接回tcp包，异步
func (l *UdpListener) rdpMakeTcpPackageSend(msg common.UDPMsg) {
	log.Infof("udp 写包 msg")
	packageChan <- msg
}

// rdpMakeTcpPackageSendBackend 异步的后台拼包
func (l *UdpListener) rdpMakeTcpPackageSendBackend() {
	for {
		var m = make(map[int]bool)
		msg := <-packageChan
		log.Infof("udp package msg= %+v", msg)
		seq := strconv.Itoa(msg.Seq)
		if msg.Count == 1 {
			l.WriteMsgToRdp(msg.Data)
			continue
		}
		status, err := cache.Get(cache_key_package + seq)
		if err != nil {
			//不存在。写入。
			m[msg.Offset] = true
			cache.SetWithExpire(cache_key_package+seq, m, 30*time.Second)
			continue
		}
		m = status.(map[int]bool)
		m[msg.Offset] = true
		cache.SetWithExpire(cache_key_package+seq, m, 30*time.Second)

		pgCache, err := cache.Get(cache_key + seq)
		var pg []byte
		if err != nil {
			pg = make([]byte, msg.Lenth)
		} else {
			pg = pgCache.([]byte)
		}

		dataLen := len(msg.Data)
		for i := 0; i < dataLen; i++ {
			pg[msg.Offset+i] = msg.Data[i]
		}
		cache.SetWithExpire(cache_key+seq, pg, 30*time.Second)
		if len(m) == msg.Count {
			//拼够包了
			l.WriteMsgToRdp(pg)
			continue
		}
	}
}
