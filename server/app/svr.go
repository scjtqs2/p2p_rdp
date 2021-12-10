package app

import (
	"encoding/json"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/scjtqs2/p2p_rdp/common"
	"github.com/scjtqs2/p2p_rdp/server/config"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type UdpListener struct {
	Port              int
	peers             *Peers //专门用于rdp转发的p2p端口的地址
	peersForSvcTrance *Peers //专门用于和svc通信的p2p端口的地址
	Cron              *cron.Cron
	//ch                chan UdpSend
}

type Peers struct {
	Mu    *sync.RWMutex
	peers map[string]*common.Peer
}

//Run 启动脚本
func (l *UdpListener) Run(config *config.ServerConfig) (err error) {
	udpAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.Host, config.Port))
	//conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: config.Port})
	conn, err := net.ListenUDP("udp", udpAddr)
	l.Port = config.Port
	if err != nil {
		log.Errorf("监听udp失败 host=%s:%d ,err=%s", config.Host, config.Port, err.Error())
		return err
	}
	//go l.sendBackend(conn)
	//l.ch = make(chan UdpSend, 128)
	log.Printf("本地地址: <%s:%d> ", config.Host, l.Port)
	l.peers = &Peers{
		peers: make(map[string]*common.Peer),
		Mu:    &sync.RWMutex{},
	}
	l.peersForSvcTrance = &Peers{
		peers: make(map[string]*common.Peer),
		Mu:    &sync.RWMutex{},
	}
	go func(conn *net.UDPConn) {
		defer conn.Close()
		for {
			data := make([]byte, common.PACKAGE_SIZE)
			n, remoteAddr, err := conn.ReadFromUDP(data)
			if err != nil {
				log.Errorf("read udp package from client faild,error %s", err.Error())
				continue
			}
			//log.Printf("<%s> %s\n", remoteAddr, data[:n])
			var udpMsg common.UDPMsg
			err = json.Unmarshal(data[:n], &udpMsg)
			if err != nil {
				log.Errorf("错误的udp包 remoteAdd=%s err=%s", remoteAddr, err.Error())
				continue
			}
			switch udpMsg.Code {
			case common.UDP_TYPE_REPORT:
				l.progressReport(udpMsg, remoteAddr)
			case common.UDP_TYPE_GET_CLIENT_IP:
				var msg common.Msg
				json.Unmarshal(udpMsg.Data, &msg)
				switch msg.Type {
				case common.CLIENT_SERVER_TYPE:
					log.Infof("group %s server type of client req addr:%s", msg.AppName, remoteAddr)
					l.progressServerClient(remoteAddr, msg, conn)
				case common.CLIENT_CLIENT_TYPE:
					log.Infof("group %s client type of client req addr:%s", msg.AppName, remoteAddr)
					l.progressClientClient(remoteAddr, msg, conn)
				default:
					log.Errorf("error type of udp req")
					continue
				}
			}
		}
	}(conn)
	l.startCron()
	return nil
}

func (l *UdpListener) progressReport(req common.UDPMsg, add net.Addr) {
	var msg common.Msg
	json.Unmarshal(req.Data, &msg)
	log.Infof("report group %s type %s req addr:%s", msg.AppName, msg.Type, add.String())
	l.checkipInListAndUpdateTime(add.String(), msg.AppName, msg.Type)
}

//progressClientClient 处理客户侧的客户端请求
func (l *UdpListener) progressClientClient(add *net.UDPAddr, req common.Msg, conn *net.UDPConn) {
	check := l.checkipInListAndUpdateTimeFroSvc(add.String(), req.AppName, req.Type)
	peers := l.PeersGet(req.AppName)
	//查找server侧的客户端地址并返回
	if peers == nil || peers.Server.Addr == "" {
		msg, _ := json.Marshal(&common.Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS,
			Res: common.Res{
				Code:    404, //不存在，404
				Message: "服务侧不在线",
			},
		})
		udpMsg, _ := json.Marshal(&common.UDPMsg{
			Code: common.UDP_TYPE_DISCOVERY,
			Data: msg,
		})
		//回一个包，确认打通udp通道。
		conn.WriteToUDP(udpMsg, add)
		//go l.WriteToUdb(udpMsg, add)
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
		udpMsg, _ := json.Marshal(&common.UDPMsg{
			Code: common.UDP_TYPE_DISCOVERY,
			Data: msg,
		})
		//发给client侧 server的ip地址。
		conn.WriteToUDP(udpMsg, add)
		//go l.WriteToUdb(udpMsg, add)
		//同时给server侧发送client的ip
		peers2 := l.Peers2Get(req.AppName)
		dstAddr, _ := net.ResolveUDPAddr("udp", peers2.Server.Addr)
		clientIp, _ := json.Marshal(peers.Client)
		msg2server, _ := json.Marshal(&common.Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS,
			Res: common.Res{
				Code:    0,
				Message: string(clientIp),
			},
		})
		msg2serverudpMsg, _ := json.Marshal(&common.UDPMsg{
			Code: common.UDP_TYPE_DISCOVERY,
			Data: msg2server,
		})
		if !check {
			msg2serverudpMsg, _ = json.Marshal(&common.UDPMsg{
				Code: common.UDP_TYPE_DISCOVERY_FROCE_P2P,
				Data: msg2server,
			})
		}
		conn.WriteToUDP(msg2serverudpMsg, dstAddr)
		//go l.WriteToUdb(msg2serverudpMsg, dstAddr)
		return
	}
}

// 处理服务侧的客户端请求
func (l *UdpListener) progressServerClient(add *net.UDPAddr, req common.Msg, conn *net.UDPConn) {
	check := l.checkipInListAndUpdateTimeFroSvc(add.String(), req.AppName, req.Type)
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
		udpMsg, _ := json.Marshal(&common.UDPMsg{
			Code: common.UDP_TYPE_DISCOVERY,
			Data: msg,
		})
		//回一个包，确认打通udp通道。
		conn.WriteToUDP(udpMsg, add)
		//go l.WriteToUdb(udpMsg, add)
		return
	} else {
		//clients有ip存在
		//直接回包client侧ip的地址列表
		clientIp, _ := json.Marshal(peers.Client)
		msg2server, _ := json.Marshal(common.Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_SERVER_WITH_CLIENT_IPS,
			Res: common.Res{
				Code:    0,
				Message: string(clientIp),
			},
		})
		msg2serverudpMsg, _ := json.Marshal(&common.UDPMsg{
			Code: common.UDP_TYPE_DISCOVERY,
			Data: msg2server,
		})
		conn.WriteToUDP(msg2serverudpMsg, add)
		//go l.WriteToUdb(msg2serverudpMsg, add)
		//对client客户端回server的ip地址。
		serverIp, _ := json.Marshal(peers.Server)
		msg2client, _ := json.Marshal(common.Msg{
			AppName: req.AppName,
			Type:    common.MESSAGE_TYPE_FOR_CLIENT_CLIENT_WITH_SERVER_IPS,
			Res: common.Res{
				Code:    0,
				Message: string(serverIp),
			},
		})
		msg2clientudpMsg, _ := json.Marshal(&common.UDPMsg{
			Code: common.UDP_TYPE_DISCOVERY,
			Data: msg2client,
		})
		if !check {
			msg2clientudpMsg, _ = json.Marshal(&common.UDPMsg{
				Code: common.UDP_TYPE_DISCOVERY_FROCE_P2P,
				Data: msg2client,
			})
		}
		cip := l.Peers2Get(req.AppName).Client
		dstAddr, _ := net.ResolveUDPAddr("udp", cip.Addr)
		conn.WriteToUDP(msg2clientudpMsg, dstAddr)
		//go l.WriteToUdb(msg2clientudpMsg, dstAddr)
	}
}

//   检查客户侧的客户端的Ip是否存在
func (l *UdpListener) checkipInListAndUpdateTime(addr, appName, clientType string) bool {
	peers := l.PeersGet(appName)
	if peers == nil {
		switch clientType {
		case common.CLIENT_CLIENT_TYPE:
			l.PeersSet(appName, &common.Peer{Client: common.Ip{Addr: addr, Time: time.Now()}})
		case common.CLIENT_SERVER_TYPE:
			l.PeersSet(appName, &common.Peer{Server: common.Ip{Addr: addr, Time: time.Now()}})
		}
		return false
	}
	switch clientType {
	case common.CLIENT_CLIENT_TYPE:
		peers.Client.Time = time.Now()
		if peers.Client.Addr == addr {
			l.PeersSet(appName, peers)
			return true
		}
		peers.Client.Addr = addr
	case common.CLIENT_SERVER_TYPE:
		peers.Server.Time = time.Now()
		if peers.Server.Addr == addr {
			//更新时间点
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

//   检查客户侧的客户端的Ip是否存在
func (l *UdpListener) checkipInListAndUpdateTimeFroSvc(addr, appName, clientType string) bool {
	peers := l.Peers2Get(appName)
	if peers == nil {
		switch clientType {
		case common.CLIENT_CLIENT_TYPE:
			l.Peers2Set(appName, &common.Peer{Client: common.Ip{Addr: addr, Time: time.Now()}, Server: common.Ip{Addr: ""}})
		case common.CLIENT_SERVER_TYPE:
			l.Peers2Set(appName, &common.Peer{Server: common.Ip{Addr: addr, Time: time.Now()}, Client: common.Ip{Addr: ""}})
		}
		return false
	}
	switch clientType {
	case common.CLIENT_CLIENT_TYPE:
		peers.Client.Time = time.Now()
		if peers.Client.Addr == addr {
			l.Peers2Set(appName, peers)
			return true
		}
		peers.Client.Addr = addr
	case common.CLIENT_SERVER_TYPE:
		peers.Server.Time = time.Now()
		if peers.Server.Addr == addr {
			//更新时间点
			l.Peers2Set(appName, peers)
			return true
		}
		peers.Server.Addr = addr
	}
	l.Peers2Set(appName, peers)
	return false
}

// PeersGet 读取peersForSvcTrance
func (l *UdpListener) Peers2Get(appName string) *common.Peer {
	l.peersForSvcTrance.Mu.RLock()
	defer l.peersForSvcTrance.Mu.RUnlock()
	return l.peersForSvcTrance.peers[appName]
}

// PeersSet 设置peersForSvcTrance
func (l *UdpListener) Peers2Set(appName string, peer *common.Peer) {
	l.peersForSvcTrance.Mu.Lock()
	defer l.peersForSvcTrance.Mu.Unlock()
	l.peersForSvcTrance.peers[appName] = peer
}

func (l *UdpListener) Peers2Count() int {
	l.peersForSvcTrance.Mu.RLock()
	defer l.peersForSvcTrance.Mu.RUnlock()
	return len(l.peersForSvcTrance.peers)
}

func (l *UdpListener) Peers2Keys() []string {
	l.peersForSvcTrance.Mu.RLock()
	defer l.peersForSvcTrance.Mu.RUnlock()
	keys := make([]string, len(l.peersForSvcTrance.peers))
	for k := range l.peersForSvcTrance.peers {
		keys = append(keys, k)
	}
	return keys
}

//
//// 异步发包 的结构体
//type UdpSend struct {
//	Addr *net.UDPAddr
//	Data []byte
//}
//
//// 将要发送的udp包写入消息队列
//func (l *UdpListener) WriteToUdb(data []byte, addr *net.UDPAddr) {
//	udpSend := UdpSend{
//		Data: data,
//		Addr: addr,
//	}
//	l.ch <- udpSend
//}
//
//// 异步消费发包，提高性能
//func (l *UdpListener) sendBackend(conn *net.UDPConn) {
//	for {
//		udpSend := <-l.ch
//		log.Printf("udp回包 toAddr=%s,data=%s", udpSend.Addr.String(), string(udpSend.Data))
//		go func(conn *net.UDPConn, send UdpSend) {
//			_, err := conn.WriteToUDP(send.Data, send.Addr)
//			if err != nil {
//				log.Errorf("回包失败 err:=%s", err.Error())
//			}
//		}(conn, udpSend)
//	}
//}
