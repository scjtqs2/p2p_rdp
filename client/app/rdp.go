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
	//go l.rdpMakeTcpPackageSendBackend()
}

func (l *UdpListener) RdpHandler() {
	var err error
	for {
		l.RdpConn, err = l.RdpListener.Accept()
		if err != nil {
			fmt.Println("accept failed, err:", err)
			continue
		}
		go l.rdpProcessTrace()
	}
}

func (l *UdpListener) rdpProcessTrace() {
	data := make([]byte, common.PACKAGE_SIZE)
	//var data []byte
	reader := bufio.NewReader(l.RdpConn)
	for {
		//l.RdpConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := reader.Read(data)
		tcpPackage := data[:n]
		if err == io.EOF {
			l.RdpConn.Close()
			break
		}
		if err != nil {
			log.Errorf("error during read: %s", err.Error())
			break
		} else {
			msg, _ := json.Marshal(&common.UDPMsg{Code: 2, Data: tcpPackage})
			//转发到远程client
			go l.WriteMsgToClient(msg)
			//go l.rdpMakeUdpPackageSend(tcpPackage)
		}
	}

}

func (l *UdpListener) rdpClientProcess() {
	var err error
	for {
		//初始化udp client
		l.RdpConn, err = net.DialTimeout("tcp", l.Conf.RemoteRdpAddr, 5*time.Second)
		if err != nil {
			log.Fatalf("init rdp client faild err=%s", err.Error())
		}
		l.rdpClientReadProcess()
	}
}

func (l *UdpListener) rdpClientReadProcess() {
	for {
		data := make([]byte, common.PACKAGE_SIZE)
		//var data []byte
		n, err := l.RdpConn.Read(data)
		if err == io.EOF {
			continue
		}
		if err != nil {
			log.Errorf("error during read: %s", err.Error())
			l.RdpConn.Close()
			return
		} else {
			recv := data[:n]
			msg, _ := json.Marshal(&common.UDPMsg{Code: common.UDP_TYPE_TRANCE, Data: recv})
			//转发到远程client
			go l.WriteMsgToClient(msg)
			//go l.rdpMakeUdpPackageSend(recv)
		}
	}
}

//rdpMakeUdpPackageSend 将tcp的包拆分成多个udp的包
func (l *UdpListener) rdpMakeUdpPackageSend(data []byte) {
	rand.Seed(time.Now().UnixNano())
	seq := rand.Intn(100000)
	packageLen := len(data)
	page := int(math.Ceil(float64(packageLen) / float64(common.PACKAGE_SIZE)))
	log.Infof("seq=%d packagelen=%d page=%d", seq, packageLen, page)
	//if packageLen == 0 {
	//	msg, _ := json.Marshal(&common.UDPMsg{Code: common.UDP_TYPE_TRANCE, Data: data, Seq: seq, Offset: 1, Count: page, Lenth: packageLen})
	//	l.WriteMsgToClient(msg)
	//	return
	//}
	for i := 1; i <= page; i++ {
		sliceStart := (i - 1) * common.PACKAGE_SIZE
		sliceEnd := sliceStart + common.PACKAGE_SIZE
		if sliceEnd > packageLen {
			sliceEnd = packageLen
		}
		msg, _ := json.Marshal(&common.UDPMsg{Code: common.UDP_TYPE_TRANCE, Data: data[sliceStart:sliceEnd], Seq: seq, Offset: i, Count: page, Lenth: packageLen})
		//log.Infof("tcp 写包 msg=%+V", msg)
		go l.WriteMsgToClient(msg)
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
		if msg.Count <= 1 {
			go l.WriteMsgToRdp(msg.Data)
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
			go l.WriteMsgToRdp(pg)
			continue
		}
	}
}
