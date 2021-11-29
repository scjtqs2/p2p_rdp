package common

import (
	"time"
)

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
	Client Ip
}

type UDPMsg struct {
	Code int64 //0:心跳 1:打洞消息 2:转发消息 3:和svc之间的通信
	Data []byte //转发/携带 的数据
}