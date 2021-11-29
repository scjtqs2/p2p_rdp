package app

import (
	"encoding/json"
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
}

type Peers struct {
	Mu    *sync.RWMutex
	peers map[string]*Peer
}

type Msg struct {
	Type    string //消息类型
	AppName string //应用类型
	Res     Res
}

type Req struct {
	AppName string //应用分类
	Type    string //rdp的类型。用来区分rdp的服务端和rdp的客户端。
	Message string //消息类型
}

type Res struct {
	Code    int64  //错误码 0成功
	Message string //消息
}

type Ip struct {
	Addr string
	Time time.Time
}

type Peer struct {
	Server  Ip
	clients []Ip
}

//Run 启动脚本
func (l *UdpListener) Run(config *config.ServerConfig) (err error) {
	l.Conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: config.Port})
	l.Port = config.Port
	if err != nil {
		log.Errorf("监听udp失败 host=%s:%d ,err=%s", config.Host, config.Port, err.Error())
		return err
	}
	log.Printf("本地地址: <%s> \n", l.Conn.LocalAddr().String())
	l.peers = Peers{
		peers: make(map[string]*Peer),
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
			var msg Req
			err = json.Unmarshal(data[:n], &msg)
			if err != nil {
				log.Errorf("错误的udp包 remoteAdd=%s err=%s", remoteAddr.String(), err.Error())
				continue
			}
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
func (l *UdpListener) progressClientClient(add *net.UDPAddr, req Req) {
	l.checkipInListAndUpdateTime(add.String(), req.AppName, req.Type)
	//if req.Message != "" {
	//	var message Msg
	//	err:=json.Unmarshal([]byte(req.Message),&message)
	//	if err == nil && message.Type==common.MESSAGE_TYPE_KEEP_ALIVE {
	//		log.Infof("remoteaddr=%s keep alive",add.String())
	//		return
	//	}
	//}
	//查找server侧的客户端地址并返回
	if l.PeersGet(req.AppName).Server.Addr == "" {
		msg, _ := json.Marshal(Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS,
			Res: Res{
				Code:    404, //不存在，404
				Message: "服务侧不在线",
			},
		})
		//回一个包，确认打通udp通道。
		go l.WriteMsgBylconn(add, msg)
		return
	} else {
		// server侧的ip存在
		ips, _ := json.Marshal([]Ip{l.PeersGet(req.AppName).Server})
		msg, _ := json.Marshal(Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS,
			Res: Res{
				Code:    0,
				Message: string(ips),
			},
		})
		//发给client侧 server的ip地址。
		go l.WriteMsgBylconn(add, msg)
		//同时给server侧发送client的ip
		serverPort, _ := strconv.Atoi(strings.Split(l.PeersGet(req.AppName).Server.Addr, ":")[1])
		srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: l.Port} // 注意端口必须固定，udp打洞，需要两侧
		dstAddr := &net.UDPAddr{IP: net.ParseIP(strings.Split(l.PeersGet(req.AppName).Server.Addr, ":")[0]), Port: serverPort}
		clientIps, _ := json.Marshal(l.PeersGet(req.AppName).clients)
		msg2server, _ := json.Marshal(Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS,
			Res: Res{
				Code:    0,
				Message: string(clientIps),
			},
		})
		go l.WriteMsg(srcAddr, dstAddr, msg2server)
		return
	}
}

// 处理服务侧的客户端请求
func (l *UdpListener) progressServerClient(add *net.UDPAddr, req Req) {
	l.checkipInListAndUpdateTime(add.String(), req.AppName, req.Type)
	//if req.Message != "" {
	//	var message Msg
	//	err:=json.Unmarshal([]byte(req.Message),&message)
	//	if err == nil && message.Type==common.MESSAGE_TYPE_KEEP_ALIVE {
	//		log.Infof("remoteaddr=%s keep alive",add.String())
	//		return
	//	}
	//}
	//查找client侧的客户端是否有地址
	peers := l.PeersGet(req.AppName)
	if peers == nil || len(peers.clients) <= 0 {
		msg, _ := json.Marshal(Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS,
			Res: Res{
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
		clientIps, _ := json.Marshal(l.PeersGet(req.AppName).clients)
		msg2server, _ := json.Marshal(Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS,
			Res: Res{
				Code:    0,
				Message: string(clientIps),
			},
		})
		go l.WriteMsgBylconn(add, msg2server)
		//对client客户端回server的ip地址。
		serverIps, _ := json.Marshal([]Ip{l.PeersGet(req.AppName).Server})
		msg2client, _ := json.Marshal(Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS,
			Res: Res{
				Code:    0,
				Message: string(serverIps),
			},
		})
		clients := l.PeersGet(req.AppName).clients
		go func() {
			srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: l.Port} // 注意端口必须固定，udp打洞，需要两侧
			for _, client := range clients {
				cip := client
				clientPort, _ := strconv.Atoi(strings.Split(cip.Addr, ":")[1])
				dstAddr := &net.UDPAddr{IP: net.ParseIP(strings.Split(cip.Addr, ":")[0]), Port: clientPort}
				l.WriteMsg(srcAddr, dstAddr, msg2client)
			}
		}()
	}
}

//   检查客户侧的客户端的Ip是否存在
func (l *UdpListener) checkipInListAndUpdateTime(addr, appName, clientType string) bool {
	if l.PeersGet(appName) == nil || len(l.PeersGet(appName).clients) <= 0 {
		switch clientType {
		case common.CLIENT_CLIENT_TYPE:
			l.PeersSet(appName, &Peer{clients: []Ip{{Addr: addr, Time: time.Now()}}})
		case common.CLIENT_SERVER_TYPE:
			l.PeersSet(appName, &Peer{Server: Ip{Addr: addr, Time: time.Now()}})
		}
		return false
	}
	switch clientType {
	case common.CLIENT_CLIENT_TYPE:
		for k, v := range l.PeersGet(appName).clients {
			if addr == v.Addr {
				//存在ip。更新时间点
				peers := l.PeersGet(appName)
				peers.clients[k].Time = time.Now()
				l.PeersSet(appName, peers)
				return true
			}
		}
		peers := l.PeersGet(appName)
		//到这里了，说明不存在
		peers.clients = append(peers.clients, Ip{Addr: addr, Time: time.Now()})
		l.PeersSet(appName, peers)
	case common.CLIENT_SERVER_TYPE:
		peers := l.PeersGet(appName)
		if peers.Server.Addr == addr {
			//更新时间点
			peers.Server.Time = time.Now()
			l.PeersSet(appName, peers)
			return true
		}
	}
	return false
}

// PeersGet 读取peers
func (l *UdpListener) PeersGet(appName string) *Peer {
	l.peers.Mu.RLock()
	defer l.peers.Mu.RUnlock()
	return l.peers.peers[appName]
}

// PeersSet 设置peers
func (l *UdpListener) PeersSet(appName string, peer *Peer) {
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
	i := 0
	for i < 3 {
		_, err := l.Conn.WriteTo(msg, add)
		if err == nil {
			return
		}
		log.Errorf("write to addr=%s faild", add.String())
		i++
	}
}

// 手动发包 3次重试
func (l *UdpListener) WriteMsg(srcAddr *net.UDPAddr, dstAddr *net.UDPAddr, msg []byte) {
	conn, _ := net.DialUDP("udp", srcAddr, dstAddr)
	i := 0
	defer conn.Close()
	for i < 3 {
		_, err := conn.Write(msg)
		if err == nil {
			return
		}
		log.Errorf("write to addr=%s faild", dstAddr.String())
		i++
	}
}
