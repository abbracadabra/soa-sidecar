package nameService

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"test/cmd/config"
	"test/utils/arrayUtil"
)

// golang包初始化方法
func init() {
	cli = createClient()
}

var (
	cli     naming_client.INamingClient
	client  = &http.Client{}
	servers = make([][3]any, 3)
)

func Subscribe(servName string, cb func(services []model.SubscribeService, err error)) {
	// Subscribe key=serviceName+groupName+cluster
	// 注意:我们可以在相同的key添加多个SubscribeCallback.
	var subscribeErr = cli.Subscribe(&vo.SubscribeParam{
		ServiceName: servName,
		GroupName:   "group-a",             // 默认值DEFAULT_GROUP
		Clusters:    []string{"cluster-a"}, // 默认值DEFAULT
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			cb(services, err)
			log.Printf("\n\n callback return services:%s \n\n")

		},
	})
	fmt.Println(subscribeErr.Error())
}

func choose() *[3]any {
	var ins *[3]any
	for _, ins = range servers {
		if ins[2] == true {
			break
		}
	}
	if ins == nil {
		ins = &servers[0]
	}
	return ins
}

// https://nacos.io/zh-cn/docs/v2/guide/user/open-api.html
func RegisterInstance(servName string, ip string, port int, tags map[string]string) error {

	var ins = choose()
	// 构建请求数据
	data := url.Values{}
	data.Set("serviceName", servName)
	data.Set("ip", ip)
	data.Set("port", strconv.Itoa(port))
	jsonTags, _ := json.Marshal(tags)
	data.Set("metadata", string(jsonTags))

	// 创建请求
	insHost := ins[0].(string) + ":" + strconv.Itoa(ins[1].(int))
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/nacos/v2/ns/instance", insHost), bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 发送请求
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		ins[2] = false
		return err
	}
	ins[2] = true

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	bodyStr := string(bodyBytes)
	success := strings.Contains(bodyStr, "success")
	if !success {
		return fmt.Errorf("register instance failed, %s", bodyStr)
	}
	return nil
}

//func Heartbeat(servName string, ip string, port int, tags map[string]string) {
//}
//
//func UpdateInstance(name string) {
//}

func GetServInfo(servName string) map[string]string {
	inf, err := cli.GetService(vo.GetServiceParam{
		ServiceName: servName,
		Clusters:    []string{"cluster-a"}, // 默认值DEFAULT
		GroupName:   "group-a",             // 默认值DEFAULT_GROUP
	})
	if err != nil {
		panic(err)
	}
	return inf.Metadata
}

func createClient() naming_client.INamingClient {
	cfg := config.GetConfig()
	sevrs := cfg.NacosServers
	// 创建clientConfig
	var clientConfig = constant.ClientConfig{
		//NamespaceId:         "e525eafa-f7d7-4029-83d9-008937f9d468", // 如果需要支持多namespace，我们可以创建多个client,它们有不同的NamespaceId。当namespace是public时，此处填空字符串。
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		//LogDir:              "/tmp/nacos/log",
		//CacheDir:            "/tmp/nacos/cache",
		//LogLevel:            "info",
	}

	// nacos服务地址
	servers = arrayUtil.MapSlice(sevrs, func(addr string) [3]any {
		part := strings.Split(addr, ":")
		ip := part[0]
		port, err := strconv.Atoi(part[1])
		if err != nil {
			panic(err)
		}
		return [3]any{ip, port, true}
	})

	// nacos客户端配置
	var serverConfigs = arrayUtil.MapSlice(sevrs, func(addr string) constant.ServerConfig {
		part := strings.Split(addr, ":")
		ip := part[0]
		port, err := strconv.Atoi(part[1])
		if err != nil {
			panic(err)
		}
		return constant.ServerConfig{
			IpAddr:      ip,
			Port:        uint64(port),
			ContextPath: "/nacos",
			Scheme:      "http",
		}
	})

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
