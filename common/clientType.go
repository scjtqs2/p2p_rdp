package common

const (
	CLIENT_SERVER_TYPE = "client_server_type" //client的服务侧请求
	CLIENT_CLIENT_TYPE = "client_client_type" //client的客户侧请求
)

const (
	UDP_TYPE_KEEP_ALIVE        = iota // 0:心跳
	UDP_TYPE_BI_DIRECTION_HOLE        // 1：打洞消息
	UDP_TYPE_TRANCE                   // 2：转发消息
	UDP_TYPE_DISCOVERY                // 3：和svc服务之间的通信
	UDP_TYPE_RDP                      // 4：udp协议的rdp数据转发
	UDP_TYPE_SEQ_RESPONSE             // 5: 回包确认包收到了
)

var PACKAGE_SIZE int = 65535 //tcp的包分割大小
