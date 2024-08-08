package cluster

import (
	"sync"
)

type LbStrategy interface {
	Choose() *Instance
}

type Instance struct {
	cluster Cluster
	IP      string
	Port    int
	// Pool    types.Pool
	Pool interface{}
	tags map[string]string
}

type Cluster struct {
	sync.Mutex
	name      string
	instances []*Instance
	lb        LbStrategy
}

func (c *Cluster) Choose() *Instance {
	return c.lb.Choose()
}

func FindByName(name string) *Cluster {
	value, _ := clusters.LoadOrStore(name, &Cluster{
		name: name,
		lb:   &RoundRobin{},
	})
	cls := value.(*Cluster)
	cls.Lock()
	defer cls.Unlock()
	if cls.instances != nil {
		return cls
	}
	// nacos todo get and listen
	// warm up conn

	cls.instances = make([]*Instance, 0) //initialized
}

var clusters sync.Map

// var clusters map[string]*Cluster = make(map[string]*Cluster)
