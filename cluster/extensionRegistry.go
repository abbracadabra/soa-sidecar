package cluster

import "test/types"

var LbFactories = map[string]LoadBalancerFactory{}
var PoolFactories = map[string]PoolFactory{}

func RegisterLoadBalancerFactory(protocol string, factory LoadBalancerFactory) {
	LbFactories[protocol] = factory
}

func GetLoadBalancerFactory(protocol string) LoadBalancerFactory {
	return LbFactories[protocol]
}

func RegisterPoolFactoryOut(protocol string, factory PoolFactory) {
	PoolFactories[protocol] = factory
}

func GetPoolFactoryOut(protocol string) PoolFactory {
	return PoolFactories[protocol]
}

type LoadBalancerFactory func(*Cluster) LoadBalancer

type PoolFactory func(*Cluster, *Instance) (types.Closable, error)
