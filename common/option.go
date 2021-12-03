package common

import (
	"net"
	"time"
)

type Msg struct {
	Type    string //消息类型
	AppName string //应用类型
	Res     Res
	Seq     string
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
	Server Ip
	Client Ip
}

type UDPMsg struct {
	Code   int        //0:心跳 1:打洞消息 2:转发消息 3:和svc之间的通信
	Data   []byte       //转发/携带 的数据
	Seq    string       //包标记
	Count  int          //当前标记的总包数量
	Offset int          //当前包的指针
	Lenth  int          //完整的包长度
	Addr   *net.UDPAddr //透传的对方的地址
}
