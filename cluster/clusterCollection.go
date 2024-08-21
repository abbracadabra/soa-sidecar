package cluster

import (
	"github.com/nacos-group/nacos-sdk-go/model"
	"sync"
	"test/grpcc"
	"test/nameService"
)

var outboundClusters sync.Map

func GetOrCreate(name string) *Cluster {
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
			cls.lb = instanceRouteByLaneCreator(cls)
		default:
			cls.lb = instanceRouteByLaneCreator(cls)
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
