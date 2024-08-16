package nameService

import (
	"fmt"
	"log"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

// var NS *NameService = createClient()

var cli naming_client.INamingClient = createClient()

// type NameService struct {
// 	cli naming_client.INamingClient
// }

func Subscribe(name string, cb func(services []model.SubscribeService, err error)) {
	// Subscribe key=serviceName+groupName+cluster
	// 注意:我们可以在相同的key添加多个SubscribeCallback.
	var subscribeErr = cli.Subscribe(&vo.SubscribeParam{
		ServiceName: "demo.go",
		GroupName:   "group-a",             // 默认值DEFAULT_GROUP
		Clusters:    []string{"cluster-a"}, // 默认值DEFAULT
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			cb(services, err)
			log.Printf("\n\n callback return services:%s \n\n")

		},
	})
	fmt.Println(subscribeErr.Error())
}

func Register(name string, ip string, port int) {

}

func createClient() naming_client.INamingClient {
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

	var namingClient, _ = clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	return namingClient
}

// SelectOneHealthyInstance将会按加权随机轮询的负载均衡策略返回一个健康的实例
// 实例必须满足的条件：health=true,enable=true and weight>0
// var instance, errf = namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
// 	ServiceName: "demo.go",
// 	GroupName:   "group-a",             // 默认值DEFAULT_GROUP
// 	Clusters:    []string{"cluster-a"}, // 默认值DEFAULT
// })
