package cluster

import (
	"strconv"
	"sync"
	"test/connPool/shared"
	"test/grpcc"
	"test/nameService"

	"github.com/nacos-group/nacos-sdk-go/model"
)

var outboundClusters sync.Map

type InstanceStatus int

const (
	REMOVED InstanceStatus = 1
)

type Instance struct {
	cluster *Cluster
	IP      string
	Port    int
	Status  InstanceStatus
	Pool    interface{}
	Meta    map[string]string
	//idc todo
}

type Cluster struct {
	sync.Mutex
	initialized bool
	servName    string
	instances   []*Instance
	lb          InstanceRouter
	poolFactory PoolFactory
	meta        map[string]string
	// connPoolingParams map[string]string
}

func (cls *Cluster) GetInstances() []*Instance {
	return cls.instances
}

// 是否要把lb放成cls的属性
func (c *Cluster) Choose(routeInfo interface{}) *Instance {
	return c.lb(routeInfo)
}

func (c *Cluster) Add(ip string, port int, meta map[string]string) error {
	c.Lock()
	defer c.Unlock()
	ins := Instance{
		cluster: c,
		IP:      ip,
		Port:    port,
		Meta:    meta,
	}
	if c.poolFactory == nil {
		_pool, err := c.poolFactory(c, &ins)
		if err != nil {
			return err
		}
		ins.Pool = _pool
	}
	c.instances = append(c.instances, &ins)
	return nil
}

func (c *Cluster) Update(services []model.SubscribeService) error {
	c.Lock()
	defer c.Unlock()
	unchanged, added, removed := diff(c.instances, services)
	//grpc strategy ??
	for _, del := range removed {
		del.Status = REMOVED
		del.Pool.(*shared.Pool).Shutdown()
	}
	for _, add := range added {
		newIns := Instance{
			cluster: c,
			IP:      add.Ip,
			Port:    int(add.Port),
			Meta:    add.Metadata,
		}
		if c.poolFactory == nil {
			_pool, err := c.poolFactory(c, &newIns)
			if err != nil {
				return err
			}
			newIns.Pool = _pool
		}
		unchanged = append(unchanged, &newIns)
	}
	c.instances = unchanged
	return nil
}

func GetOrCreate(name string, routeCreator InstanceRouterCreator) *Cluster {
	var cls *Cluster
	_value, ok := outboundClusters.Load(name)
	if !ok {
		var servMeta = nameService.GetServInfo(name)
		cls = &Cluster{
			servName:  name,
			instances: make([]*Instance, 0),
			meta:      servMeta,
		}
		ptc := servMeta["protocol"]
		// caller without prior knowledge,e.g. read pre config cluster
		switch ptc {
		case "grpc":
			cls.poolFactory = grpcc.PoolFactoryOut
		}
		if routeCreator != nil {
			cls.lb = routeCreator(cls)
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
			// Shared service: Update meta with Metadata from new service
			cachedInstance.Meta = svc.Metadata
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

type InstanceRouterCreator func(*Cluster) InstanceRouter

type PoolFactory func(*Cluster, *Instance) (interface{}, error)

type InstanceRouter func(interface{}) *Instance
