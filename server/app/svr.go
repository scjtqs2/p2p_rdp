package app

import (
	"encoding/json"
	"github.com/bluele/gcache"
	"github.com/robfig/cron/v3"
	"github.com/scjtqs2/p2p_rdp/common"
	"github.com/scjtqs2/p2p_rdp/server/config"
	log "github.com/sirupsen/logrus"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type UdpListener struct {
	Port  int
	Conn  *net.UDPConn
	peers Peers
	Cron  *cron.Cron
	Cache gcache.Cache
}

type Peers struct {
	Mu    *sync.RWMutex
	peers map[string]*common.Peer
}

//Run 启动脚本
func (l *UdpListener) Run(config *config.ServerConfig) (err error) {
	l.Conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: config.Port})
	l.Port = config.Port
	if err != nil {
		log.Errorf("监听udp失败 host=%s:%d ,err=%s", config.Host, config.Port, err.Error())
		return err
	}
	l.Cache = gcache.New(200).LRU().Build()
	log.Printf("本地地址: <%s> \n", l.Conn.LocalAddr().String())
	l.peers = Peers{
		peers: make(map[string]*common.Peer),
		Mu:    &sync.RWMutex{},
	}
	go func() {
		for {
			data := make([]byte, 1024)
			n, remoteAddr, err := l.Conn.ReadFromUDP(data)
			if err != nil {
				log.Errorf("error during read: %s", err)
				continue
			}
			log.Printf("<%s> %s\n", remoteAddr.String(), data[:n])
			var msg common.Msg
			err = json.Unmarshal(data[:n], &msg)
			if err != nil {
				log.Errorf("错误的udp包 remoteAdd=%s err=%s", remoteAddr.String(), err.Error())
				continue
			}
			l.returnSeq(msg.Seq, remoteAddr)
			if l.checkSeq(msg.Seq) {
				continue
			}
			l.setSeq(msg.Seq)
			switch msg.Type {
			case common.CLIENT_SERVER_TYPE:
				log.Infof("server type of client req addr:%s", remoteAddr.String())
				l.progressServerClient(remoteAddr, msg)
			case common.CLIENT_CLIENT_TYPE:
				log.Infof("client type of client req addr:%s", remoteAddr.String())
				l.progressClientClient(remoteAddr, msg)
			default:
				log.Errorf("error type of udp req")
				continue
			}
		}
	}()
	l.startCron()
	return nil
}

//progressClientClient 处理客户侧的客户端请求
func (l *UdpListener) progressClientClient(add *net.UDPAddr, req common.Msg) {
	l.checkipInListAndUpdateTime(add.String(), req.AppName, req.Type)
	//查找server侧的客户端地址并返回
	if l.PeersGet(req.AppName).Server.Addr == "" {
		msg, _ := json.Marshal(common.Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS,
			Res: common.Res{
				Code:    404, //不存在，404
				Message: "服务侧不在线",
			},
		})
		//回一个包，确认打通udp通道。
		go l.WriteMsgBylconn(add, msg)
		return
	} else {
		// server侧的ip存在
		serverIp, _ := json.Marshal(l.PeersGet(req.AppName).Server)
		msg, _ := json.Marshal(common.Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS,
			Res: common.Res{
				Code:    0,
				Message: string(serverIp),
			},
		})
		//发给client侧 server的ip地址。
		go l.WriteMsgBylconn(add, msg)
		//同时给server侧发送client的ip
		serverPort, _ := strconv.Atoi(strings.Split(l.PeersGet(req.AppName).Server.Addr, ":")[1])
		dstAddr := &net.UDPAddr{IP: net.ParseIP(strings.Split(l.PeersGet(req.AppName).Server.Addr, ":")[0]), Port: serverPort}
		clientIp, _ := json.Marshal(l.PeersGet(req.AppName).Client)
		msg2server, _ := json.Marshal(common.Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS,
			Res: common.Res{
				Code:    0,
				Message: string(clientIp),
			},
		})
		go l.WriteMsg(dstAddr, msg2server)
		return
	}
}

// 处理服务侧的客户端请求
func (l *UdpListener) progressServerClient(add *net.UDPAddr, req common.Msg) {
	l.checkipInListAndUpdateTime(add.String(), req.AppName, req.Type)
	//查找client侧的客户端是否有地址
	peers := l.PeersGet(req.AppName)
	if peers == nil || peers.Client.Addr == "" {
		msg, _ := json.Marshal(common.Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS,
			Res: common.Res{
				Code:    404, //不存在，404
				Message: "客户侧不在线",
			},
		})
		//回一个包，确认打通udp通道。
		go l.WriteMsgBylconn(add, msg)
		return
	} else {
		//clients有ip存在
		//直接回包client侧ip的地址列表
		clientIp, _ := json.Marshal(l.PeersGet(req.AppName).Client)
		msg2server, _ := json.Marshal(common.Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS,
			Res: common.Res{
				Code:    0,
				Message: string(clientIp),
			},
		})
		go l.WriteMsgBylconn(add, msg2server)
		//对client客户端回server的ip地址。
		serverIp, _ := json.Marshal(l.PeersGet(req.AppName).Server)
		msg2client, _ := json.Marshal(common.Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS,
			Res: common.Res{
				Code:    0,
				Message: string(serverIp),
			},
		})
		client := l.PeersGet(req.AppName).Client
		go func() {
			cip := client
			clientPort, _ := strconv.Atoi(strings.Split(cip.Addr, ":")[1])
			dstAddr := &net.UDPAddr{IP: net.ParseIP(strings.Split(cip.Addr, ":")[0]), Port: clientPort}
			l.WriteMsg(dstAddr, msg2client)
		}()
	}
}

//   检查客户侧的客户端的Ip是否存在
func (l *UdpListener) checkipInListAndUpdateTime(addr, appName, clientType string) bool {
	if l.PeersGet(appName) == nil {
		switch clientType {
		case common.CLIENT_CLIENT_TYPE:
			l.PeersSet(appName, &common.Peer{Client: common.Ip{Addr: addr, Time: time.Now()}})
		case common.CLIENT_SERVER_TYPE:
			l.PeersSet(appName, &common.Peer{Server: common.Ip{Addr: addr, Time: time.Now()}})
		}
		return false
	}
	peers := l.PeersGet(appName)
	switch clientType {
	case common.CLIENT_CLIENT_TYPE:
		if peers.Client.Addr == addr {
			peers.Client.Time = time.Now()
			l.PeersSet(appName, peers)
			return true
		}
		peers.Client.Addr = addr
	case common.CLIENT_SERVER_TYPE:
		if peers.Server.Addr == addr {
			//更新时间点
			peers.Server.Time = time.Now()
			l.PeersSet(appName, peers)
			return true
		}
		peers.Server.Addr = addr
	}
	l.PeersSet(appName, peers)
	return false
}

// PeersGet 读取peers
func (l *UdpListener) PeersGet(appName string) *common.Peer {
	l.peers.Mu.RLock()
	defer l.peers.Mu.RUnlock()
	return l.peers.peers[appName]
}

// PeersSet 设置peers
func (l *UdpListener) PeersSet(appName string, peer *common.Peer) {
	l.peers.Mu.Lock()
	defer l.peers.Mu.Unlock()
	l.peers.peers[appName] = peer
}

func (l *UdpListener) PeersCount() int {
	l.peers.Mu.RLock()
	defer l.peers.Mu.RUnlock()
	return len(l.peers.peers)
}

func (l *UdpListener) PeersKeys() []string {
	l.peers.Mu.RLock()
	defer l.peers.Mu.RUnlock()
	keys := make([]string, len(l.peers.peers))
	for k := range l.peers.peers {
		keys = append(keys, k)
	}
	return keys
}

// 用l的conn回包 3次重试
func (l *UdpListener) WriteMsgBylconn(add *net.UDPAddr, msg []byte) {
	rsp, _ := json.Marshal(&common.UDPMsg{
		Code: common.UDP_TYPE_DISCOVERY,
		Data: msg,
	})
	i := 0
	for i < 3 {
		_, err := l.Conn.WriteTo(rsp, add)
		if err == nil {
			return
		}
		log.Errorf("write to addr=%s faild", add.String())
		i++
	}
}

// 手动发包 3次重试
func (l *UdpListener) WriteMsg(dstAddr *net.UDPAddr, msg []byte) {
	rsp, _ := json.Marshal(&common.UDPMsg{
		Code: common.UDP_TYPE_DISCOVERY,
		Data: msg,
	})
	i := 0
	for i < 3 {
		_, err := l.Conn.WriteToUDP(rsp, dstAddr)
		if err == nil {
			return
		}
		log.Errorf("write to addr=%s faild", dstAddr.String())
		i++
	}
}
