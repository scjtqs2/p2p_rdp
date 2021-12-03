package main

import (
	"context"
	"flag"
	"github.com/scjtqs2/p2p_rdp/client/app"
	"github.com/scjtqs2/p2p_rdp/client/config"
	"github.com/scjtqs2/utils/util"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
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
	LogLevel := log.InfoLevel
	if debug {
		log.SetReportCaller(true)
		LogLevel = log.DebugLevel
	}
	log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05"}) //使用json的格式
	log.SetLevel(LogLevel)
}

func main() {
	if h {
		help()
	}
	if d {
		util.Daemon()
	}
	ctx:=context.Background()
	log.WithContext(ctx)
	conf := config.GetConfigFronPath(configPath)
	conf.Save(configPath)
	log.Infof("welcome to use p2p_rdp client  by scjtqs  https://github.com/scjtqs2/p2p_rdp/client %s,build in %s", Version, Build)
	udplistener := &app.UdpListener{}
	udplistener.Run(ctx,conf)
	defer func() {
		if udplistener.LocalConn != nil {
			udplistener.LocalConn.Close()
		}
	}()
	defer udplistener.Cron.Stop()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	log.Infof("初始化完成")
	<-c
}

// help cli命令行-h的帮助提示
func help() {
	log.Infof(`p2p_rdp client
version: %s
built-on: %s

Usage:

client [OPTIONS]

Options:
`, Version, Build)
	flag.PrintDefaults()
	os.Exit(0)
}
