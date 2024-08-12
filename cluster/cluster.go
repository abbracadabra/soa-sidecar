package cluster

import (
	"context"
	"strconv"
	"sync"
	"test/codec"
	"test/nameService"
	"time"

	"test/connPool2/shared"

	"github.com/nacos-group/nacos-sdk-go/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var outboundClusters sync.Map
var inboundClusters sync.Map

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

func (c *Cluster) addInstance(ip string, port int, tags map[string]string) {
	c.Lock()
	defer c.Unlock()
	ins := Instance{
		cluster: c,
		IP:      ip,
		Port:    port,
		tags:    tags,
	}
	ins.Pool = createPool(&ins)
	c.instances = append(c.instances, &ins)
}

func (c *Cluster) Update(services []model.SubscribeService) {
	c.Lock()
	defer c.Unlock()
	unchanged, added, removed := diff(c.instances, services)
	//grpc strategy ??
	for _, del := range removed {
		del.Pool.(*shared.Pool).Shutdown()
	}
	for _, add := range added {
		newIns := Instance{
			cluster: c,
			IP:      add.Ip,
			Port:    int(add.Port),
			tags:    add.Metadata,
		}
		newIns.Pool = createPool(&newIns)
		unchanged = append(unchanged, &newIns)
	}
	c.instances = unchanged
}

func FindByName(name string, outbound bool) *Cluster {
	if outbound {
		value, _ := outboundClusters.LoadOrStore(name, &Cluster{
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
			cls.Update(services)
		})
		cls.initialized = true
		return cls
	} else {
		value, _ := inboundClusters.LoadOrStore(name, &Cluster{
			name:        name,
			lb:          &RoundRobin{},
			instances:   make([]*Instance, 0),
			initialized: true,
		})
		return value.(*Cluster)
	}
}

func createPool(ins *Instance) interface{} {
	// todo get service info/config
	p := shared.NewPool(1, 1, 9999, time.Minute*1, func() (*grpc.ClientConn, func(), error) {
		ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
		cc, err := grpc.DialContext(ctx, ins.IP+":"+strconv.Itoa(ins.Port), grpc.WithCodec(codec.Codec()), grpc.WithInsecure(), grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true, // 允许没有活跃流的心跳
		}))
		if err != nil {
			return nil, nil, err
		}

		return cc, func() {
			cc.Close()
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
