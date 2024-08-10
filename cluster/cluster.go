package cluster

import (
	"context"
	"strconv"
	"sync"
	"test/nameService"
	"time"

	"test/connPool2/shared"

	"github.com/nacos-group/nacos-sdk-go/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var clusters sync.Map

type LbStrategy interface {
	Choose() *Instance
}

type Instance struct {
	cluster *Cluster
	IP      string
	Port    int
	Pool    interface{}
	tags    map[string]string
}

type Cluster struct {
	sync.Mutex
	initialized bool
	name        string
	instances   []*Instance
	lb          LbStrategy
}

func (c *Cluster) Choose() *Instance {
	return c.lb.Choose()
}

func FindByName(name string) *Cluster {
	value, _ := clusters.LoadOrStore(name, &Cluster{
		name:      name,
		lb:        &RoundRobin{},
		instances: make([]*Instance, 0),
	})
	cls := value.(*Cluster)
	cls.Lock()
	defer cls.Unlock()
	if cls.initialized {
		return cls
	}
	nameService.Subscribe(name, func(services []model.SubscribeService, err error) {

		unchanged, added, removed := diff(cls.instances, services)
		//grpc strategy ??
		for _, del := range removed {
			del.Pool.(*shared.Pool).Shutdown()
		}
		for _, add := range added {
			newIns := Instance{
				cluster: cls,
				IP:      add.Ip,
				Port:    int(add.Port),
				tags:    add.Metadata,
			}
			newIns.Pool = createPool(&newIns)
			unchanged = append(unchanged, &newIns)
		}
	})
	cls.initialized = true
	return cls
}

func createPool(ins *Instance) interface{} {
	// todo get service info/config
	p := shared.NewPool(1, 1, 9999, time.Minute*1, func() (*grpc.ClientConn, func(), error) {
		ctx, cancel := context.WithCancel(context.Background())
		cc, err := grpc.DialContext(ctx, ins.IP+":"+strconv.Itoa(ins.Port), grpc.WithCodec(Codec()), grpc.WithInsecure(), grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true, // 允许没有活跃流的心跳
		}))
		if err != nil {
			return nil, nil, err
		}

		return cc, func() {
			//cc.Close():直接关闭连接： 这个方法会直接关闭底层的 gRPC 连接，不再接收或发送任何消息。需要注意： 如果有正在进行的 RPC 调用，可能会导致这些调用失败。
			//如果您希望正在进行的 RPC 调用能够正常结束，再关闭连接，那么 ctx.Cancel() 是一个更好的选择。
			cancel()
		}, nil

	})
	return p
}

func diff(cached []*Instance, latest []model.SubscribeService) ([]*Instance, []*model.SubscribeService, []*Instance) {
	shared := []*Instance{}
	newServices := []*model.SubscribeService{}
	removed := []*Instance{}

	// Create a map for quick lookup of cached services by IP and Port
	cachedMap := make(map[string]*Instance)
	for _, instance := range cached {
		key := instance.IP + ":" + strconv.Itoa(instance.Port)
		cachedMap[key] = instance
	}

	// Iterate through the latest services
	latestMap := make(map[string]bool)
	for _, svc := range latest {
		key := svc.Ip + ":" + strconv.Itoa(int(svc.Port))
		latestMap[key] = true

		if cachedInstance, exists := cachedMap[key]; exists {
			// Shared service: Update tags with Metadata from new service
			cachedInstance.tags = svc.Metadata
			shared = append(shared, cachedInstance)
		} else {
			// New service not in cached
			newServices = append(newServices, &svc)
		}
	}

	// Find removed services
	for _, instance := range cached {
		key := instance.IP + ":" + strconv.Itoa(instance.Port)
		if !latestMap[key] {
			removed = append(removed, instance)
		}
	}

	return shared, newServices, removed
}

// var clusters map[string]*Cluster = make(map[string]*Cluster)
