package config

import (
	"errors"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type bindAddr struct {
	Ip   string
	Port int
}

type bindProtocol struct {
	bindAddr
	Transparent bool
	Secure      bool
	Protocol    string
	Outbound    bool
}

type Config struct {
	Console                 *bindAddr
	InboundTransparentAddr  *bindAddr
	OutboundTransparentAddr *bindAddr
	OutboundProxy           []bindProtocol
}

func ReadAsConfig(path string) (Config, error) {

	if path == "" {
		return Config{}, errors.New("配置文件不能为空")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("读取文件失败: %v", err)
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("解析 YAML 失败: %v", err)
	}
	return config, nil
}
