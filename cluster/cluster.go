package cluster

import (
	"github.com/nacos-group/nacos-sdk-go/model"
	"strconv"
	"sync"
	"test/types"
)

type InstanceStatus int

const (
	REMOVED InstanceStatus = 1
)

type Instance struct {
	cluster    *Cluster
	IP         string
	Port       int
	Status     InstanceStatus
	Pool       types.Closable
	Meta       map[string]string
	statistics map[string]any
}

func (ins *Instance) onRemoved() {
	if ins.Pool != nil {
		ins.Pool.Close()
	}
}

type Cluster struct {
	sync.Mutex
	servName    string
	instances   []*Instance
	lb          LoadBalancer
	poolFactory PoolFactory
	meta        map[string]string
}

func (cls *Cluster) GetInstances() []*Instance {
	return cls.instances
}

func (c *Cluster) Choose(routeInfo interface{}) *Instance {
	return c.lb.pick(routeInfo)
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
	if c.poolFactory != nil {
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
	for _, del := range removed {
		del.Status = REMOVED
		del.onRemoved()
	}
	for _, add := range added {
		newIns := Instance{
			cluster: c,
			IP:      add.Ip,
			Port:    int(add.Port),
			Meta:    add.Metadata,
		}
		if c.poolFactory != nil {
			_pool, err := c.poolFactory(c, &newIns)
			if err != nil {
				return err
			}
			newIns.Pool = _pool
		}
		unchanged = append(unchanged, &newIns)
	}
	c.instances = unchanged
	c.lb.onClusterInstanceChanged() //通知loadbalancer
	return nil
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

type LoadBalancer interface {
	pick(routeInfo interface{}) *Instance
	onClusterInstanceChanged()
}
