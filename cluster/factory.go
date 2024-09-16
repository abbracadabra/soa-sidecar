package cluster

import (
	"github.com/nacos-group/nacos-sdk-go/model"
	"test/nameService"
)

func instantCreate(name string) (*Cluster, error) {
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
}

//
//type OnDemandFactory struct {
//	registrar func(string, *Cluster)
//}
//
//func (f *OnDemandFactory) SetRegistrar(registrar func(string, *Cluster)) {
//	f.registrar = registrar
//}

//func (*OnDemandFactory) GetOrCreate(name string) (*Cluster, error) {
//	var servMeta = nameService.GetServInfo(name)
//	var cls = &Cluster{
//		servName:  name,
//		instances: make([]*Instance, 0),
//		meta:      servMeta,
//	}
//	protocol := servMeta["protocol"]
//	// 太多的话改成customizer
//	cls.poolFactory = GetPoolFactoryOut(protocol)
//	cls.lb = GetLoadBalancerFactory(protocol)(cls)
//	nameService.Subscribe(name, func(services []model.SubscribeService, err error) {
//		cls.Update(services)
//	})
//	return cls, nil
//}
