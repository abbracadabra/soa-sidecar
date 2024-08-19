package cluster

import (
	"strconv"
	"sync"
	"test/connPool/shared"
	"test/nameService"

	"github.com/nacos-group/nacos-sdk-go/model"
)

var outboundClusters sync.Map

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
	servName    string
	instances   []*Instance
	lb          LbStrategy
	poolFactory PoolFactory // big TODO
	meta        map[string]string
	// connPoolingParams map[string]string
}

func (c *Cluster) Choose() *Instance {
	return c.lb.Choose()
}

func (c *Cluster) Add(ip string, port int, tags map[string]string) error {
	c.Lock()
	defer c.Unlock()
	ins := Instance{
		cluster: c,
		IP:      ip,
		Port:    port,
		tags:    tags,
	}
	_pool, err := c.poolFactory(c, &ins)
	if err != nil {
		return err
	}
	ins.Pool = _pool
	c.instances = append(c.instances, &ins)
	return nil
}

func (c *Cluster) Update(services []model.SubscribeService) error {
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
		_pool, err := c.poolFactory(c, &newIns)
		if err != nil {
			return err
		}
		newIns.Pool = _pool
		unchanged = append(unchanged, &newIns)
	}
	c.instances = unchanged
	return nil
}

func GetOrCreate(name string, pf PoolFactory) *Cluster {
	var cls *Cluster
	_value, ok := outboundClusters.Load(name)
	if !ok {
		var servMeta = make(map[string]string)
		cls := &Cluster{
			servName:    name,
			instances:   make([]*Instance, 0),
			meta:        servMeta,
			poolFactory: pf,
		}
		if servMeta["lb"] == Robin {
			cls.lb = &RoundRobin{cls: cls}
		} else {
			cls.lb = &RoundRobin{cls: cls}
		}
		_value, _ := outboundClusters.LoadOrStore(name, cls)
		cls = _value.(*Cluster)
	} else {
		cls = _value.(*Cluster)
	}
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

type PoolFactory func(*Cluster, *Instance) (interface{}, error)
