package main

import (
	"test/cmd/config"
)

func main() {

	cfg, err := config.GetOrReadConfig()
	if err != nil {
		panic(err)
	}
	//控制台
	startConsoleServer(cfg.Console.Ip, cfg.Console.Port)
	//开启透明代理，出流量
	if cfg.OutboundTransparent {
		startTransparentOut(cfg.OutboundTransparentIp, cfg.OutboundTransparentPort)
	}
	//开启透明代理，入流量
	if cfg.InboundTransparent {
		startTransparentIn(cfg.InboundTransparentIp, cfg.InboundTransparentPort)
	}
	// 协议代理
	if cfg.OutboundProxy != nil {
		for _, v := range cfg.OutboundProxy {
			serveProtocolOut(cfg.OutboundTransparent, v.Ip, v.Port, v.Secure, v.Protocol)
		}
	}
}
