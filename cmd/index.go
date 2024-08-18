package main

import (
	"flag"
	"test/config"
)

func main() {

	path := flag.String("c", "", "配置文件路径")

	flag.Parse()

	var cfg config.Config
	cfg, err := config.ReadAsConfig(*path)
	if err != nil {
		panic(err)
	}

	//控制台
	startConsoleServer(cfg.Console.Ip, cfg.Console.Port)
	//开启透明代理，出流量
	if cfg.OutboundTransparentAddr != nil {
		startTransparentOut(cfg.OutboundTransparentAddr.Ip, cfg.OutboundTransparentAddr.Port)
	}
	//开启透明代理，入流量
	if cfg.InboundTransparentAddr != nil {
		startTransparentIn(cfg.InboundTransparentAddr.Ip, cfg.InboundTransparentAddr.Port)
	}
	// 协议代理
	if cfg.OutboundProxy != nil {
		for _, v := range cfg.OutboundProxy {
			serveProtocolOut(v.Transparent, v.Ip, v.Port, v.Secure, v.Protocol)
		}
	}
}
