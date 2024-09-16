package cluster

import (
	"math/rand"
	"sync"
	"test/utils/swimlane"
)

type RouteInfo struct {
	Color string
	Key   string
}

// MyLb 同机房  同泳道  路由hashkey
type MyLb struct {
	LoadBalancer
	instances []*Instance
	cls       *Cluster
	stickyMap sync.Map
}

func (m *MyLb) onClusterInstanceChanged() {
	m.instances = m.cls.GetInstances()
}

func (m *MyLb) pick(routeInfo interface{}) *Instance {
	rf := routeInfo.(*RouteInfo)
	bind, sticky := m.stickyMap.Load(rf.Key)
	if sticky {
		ins := bind.(*Instance)
		if ins.Status == REMOVED {
			m.stickyMap.Delete(rf.Key)
		} else {
			return ins
		}
	}
	// lane
	filtered := swimlane.FilterLane(m.cls.GetInstances(), rf.Color)
	// idc todo

	n := len(filtered)
	if n == 0 {
		return nil
	}
	i := rand.Intn(n)
	chosen := filtered[i]
	if sticky {
		m.stickyMap.Store(rf.Key, chosen)
	}
	return chosen
}

func NewDefaultLoadBalancer(cls *Cluster) LoadBalancer {
	return &MyLb{
		instances: cls.instances,
		cls:       cls,
	}
}
