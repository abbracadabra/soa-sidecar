package cluster

import (
	"github.com/nacos-group/nacos-sdk-go/model"
	"golang.org/x/sync/singleflight"
	"test/nameService"
)

var outboundClusters singleflight.Group

func GetOrCreate(name string) *Cluster {
	result, _, _ := outboundClusters.Do(name, func() (interface{}, error) {
		var servMeta = nameService.GetServInfo(name)
		var cls = &Cluster{
			servName:  name,
			instances: make([]*Instance, 0),
			meta:      servMeta,
		}
		protocol := servMeta["protocol"]
		// 太多的话改成customizer
		cls.poolFactory = GetPoolFactoryOut(protocol)
		cls.lb = GetLoadBalancerFactory(protocol)(cls)
		nameService.Subscribe(name, func(services []model.SubscribeService, err error) {
			cls.Update(services)
		})
		return cls, nil
	})
	return result.(*Cluster)
}
