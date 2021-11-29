package config

type ClientConfig struct {
	ServerHost string //服务器地址
	ServerPort int64 //服务端
	Type string //rdp的服务端还是客户端
}
