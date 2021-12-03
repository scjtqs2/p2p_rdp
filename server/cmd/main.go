package main

import (
	"flag"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/scjtqs2/p2p_rdp/server/app"
	"github.com/scjtqs2/p2p_rdp/server/config"
	"github.com/scjtqs2/utils/util"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"os"
	"os/signal"
	"path"
	"time"
)

var (
	h          bool
	d          bool
	Version    = "v1.0.0"
	Build      string
	configPath = "config.yml"
)

func init() {
	var debug bool
	flag.BoolVar(&d, "d", false, "running as a daemon")
	flag.BoolVar(&h, "h", false, "this help")
	flag.StringVar(&configPath, "c", "config.yml", "config file path default is config.yml")
	flag.Parse()
	logFormatter := &easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "[%time%] [%lvl%]: %msg% \n",
	}
	w, err := rotatelogs.New(path.Join("logs", "%Y-%m-%d.log"), rotatelogs.WithRotationTime(time.Hour*24))
	if err != nil {
		log.Errorf("rotatelogs init err: %v", err)
		panic(err)
	}
	LogLevel := "info"
	if debug {
		log.SetReportCaller(true)
		LogLevel = "debug"
	}
	log.AddHook(util.NewLocalHook(w, logFormatter, util.GetLogLevel(LogLevel)...))
}

func main() {
	if h {
		help()
	}
	if d {
		util.Daemon()
	}
	conf := config.GetConfigFronPath(configPath)
	conf.Save(configPath)
	log.Infof("welcome to use p2p_rdp server  by scjtqs  https://github.com/scjtqs2/p2p_rdp/server %s,build in %s", Version, Build)
	udplistener := &app.UdpListener{}
	udplistener.Run(conf)
	//defer udplistener.Conn.Close()
	defer udplistener.Cron.Stop()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	log.Infof("初始化完成")
	<-c
}

// help cli命令行-h的帮助提示
func help() {
	log.Infof(`p2p_rdp service
version: %s
built-on: %s

Usage:

server [OPTIONS]

Options:
`, Version, Build)
	flag.PrintDefaults()
	os.Exit(0)
}
