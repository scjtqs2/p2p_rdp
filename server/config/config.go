package config

import (
	"github.com/go-yaml/yaml"
	"github.com/scjtqs2/utils/util"
)

type ServerConfig struct {
	Host string
	Port int
}

// 通过路径获取配置信息
func GetConfigFronPath(c string) *ServerConfig {
	conf := &ServerConfig{}
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

func defaultConf() *ServerConfig {
	return &ServerConfig{
		Host: "0.0.0.0",
		Port: 30124,
	}
}

// 从环境变量中替换配置文件
func parseConfFromEnv(c *ServerConfig) *ServerConfig {
	//todu
	return c
}

// 保存配置文件
func (c *ServerConfig) Save(p string) error {
	s, _ := yaml.Marshal(c)
	return util.WriteAllText(p, string(s))
}
