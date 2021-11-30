package common

const (
	CLIENT_SERVER_TYPE         = "client_server_type" //client的服务侧请求
	CLIENT_CLIENT_TYPE         = "client_client_type" //client的客户侧请求
	UDP_TYPE_KEEP_ALIVE        = 0                    // 0:心跳
	UDP_TYPE_BI_DIRECTION_HOLE = 1                    // 1：打洞消息
	UDP_TYPE_TRANCE            = 2                    // 2：转发消息
	UDP_TYPE_DISCOVERY         = 3                    // 3：和svc服务之间的通信
)

var PACKAGE_SIZE int = 5000 //tcp的包分割大小
