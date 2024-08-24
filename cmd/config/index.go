package config

import (
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

var cfg *Config

type bindAddr struct {
	Ip   string
	Port int
}

type bindProtocol struct {
	bindAddr
	//Transparent bool
	Secure   bool
	Protocol string
	Outbound bool
}

type Config struct {
	Console       *bindAddr
	OutboundProxy []bindProtocol

	//透明代理
	InboundTransparent     bool
	InboundTransparentIp   string
	InboundTransparentPort int

	//透明代理
	OutboundTransparent     bool
	OutboundTransparentIp   string
	OutboundTransparentPort int

	NacosServers []string `ip:port`
}

func init() {
	path := flag.String("c", "", "配置文件路径")
	flag.Parse()

	if *path == "" {
		log.Fatalf("配置文件不能为空")
		os.Exit(1)
		return
	}
	data, err := os.ReadFile(*path)
	if err != nil {
		log.Fatalf("读取文件失败: %v", err)
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("解析 YAML 失败: %v", err)
	}
	cfg = &config
}

func GetConfig() Config {
	return *cfg
}
