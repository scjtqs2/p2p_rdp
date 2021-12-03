package config

import (
	"github.com/go-yaml/yaml"
	"github.com/scjtqs2/p2p_rdp/common"
	"github.com/scjtqs2/utils/util"
)

type ClientConfig struct {
	ServerHost             string //服务器地址
	ServerPort             int    //服务端
	Type                   string //rdp的服务端还是客户端
	ClientPortFroSvc       int    //客户端开的端口 为了和svc进行p2p通信
	ClientPortForP2PTrance int    //客户端开的端口，为了rdp的p2p通信
	RdpP2pPort             int    //rdp的转发请求端口
	AppName                string //当前的服务组名称。client侧和server侧必须有相同的appName才能匹配上。
	RemoteRdpAddr          string //要控制的rdp的地址
}

// 通过路径获取配置信息
func GetConfigFronPath(c string) *ClientConfig {
	conf := &ClientConfig{}
	if !util.PathExists(c) {
		conf = defaultConf()
	} else {
		err := yaml.Unmarshal([]byte(util.ReadAllText(c)), conf)
		if err != nil {
			conf = defaultConf()
		}
	}
	return parseConfFromEnv(conf)
}

func defaultConf() *ClientConfig {
	return &ClientConfig{
		ServerHost:             "1.1.1.1",
		ServerPort:             30124,
		Type:                   common.CLIENT_CLIENT_TYPE,
		AppName:                "rdp-p2p",
		RdpP2pPort:             30122,
		ClientPortForP2PTrance: 30123,
		ClientPortFroSvc:       30124,
		RemoteRdpAddr:          "127.0.0.1:3389",
	}
}

// 从环境变量中替换配置文件
func parseConfFromEnv(c *ClientConfig) *ClientConfig {
	//todo
	return c
}

// 保存配置文件
func (c *ClientConfig) Save(p string) error {
	s, _ := yaml.Marshal(c)
	return util.WriteAllText(p, string(s))
}
