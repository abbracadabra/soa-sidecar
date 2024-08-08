package nacosClient

import (
	"log"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

// 创建clientConfig
var clientConfig = constant.ClientConfig{
	NamespaceId:         "e525eafa-f7d7-4029-83d9-008937f9d468", // 如果需要支持多namespace，我们可以创建多个client,它们有不同的NamespaceId。当namespace是public时，此处填空字符串。
	TimeoutMs:           5000,
	NotLoadCacheAtStart: true,
	LogDir:              "/tmp/nacos/log",
	CacheDir:            "/tmp/nacos/cache",
	LogLevel:            "debug",
}

// 至少一个ServerConfig
var serverConfigs = []constant.ServerConfig{
	{
		IpAddr:      "console1.nacos.io",
		ContextPath: "/nacos",
		Port:        80,
		Scheme:      "http",
	},
	{
		IpAddr:      "console2.nacos.io",
		ContextPath: "/nacos",
		Port:        80,
		Scheme:      "http",
	},
}

var namingClient, err = clients.NewNamingClient(
	vo.NacosClientParam{
		ClientConfig:  &clientConfig,
		ServerConfigs: serverConfigs,
	},
)

// Subscribe key=serviceName+groupName+cluster
// 注意:我们可以在相同的key添加多个SubscribeCallback.
var errx = namingClient.Subscribe(&vo.SubscribeParam{
	ServiceName: "demo.go",
	GroupName:   "group-a",             // 默认值DEFAULT_GROUP
	Clusters:    []string{"cluster-a"}, // 默认值DEFAULT
	SubscribeCallback: func(services []model.SubscribeService, err error) {
		log.Printf("\n\n callback return services:%s \n\n")
	},
})

// SelectOneHealthyInstance将会按加权随机轮询的负载均衡策略返回一个健康的实例
// 实例必须满足的条件：health=true,enable=true and weight>0
var instance, errf = namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
	ServiceName: "demo.go",
	GroupName:   "group-a",             // 默认值DEFAULT_GROUP
	Clusters:    []string{"cluster-a"}, // 默认值DEFAULT
})
