package cluster

import "test/connPool2/types"

type LbStrategy interface {
	Choose() *Instance
}

type Instance struct {
	cluster Cluster
	IP      string
	Port    int
	Pool    *types.Pool
	tags    *map[string]string
}

type Cluster struct {
	name      string
	instances *[]Instance
	Lb        LbStrategy
}

func FindByName(name string) *Cluster {

	// already
	c := clusters[name]
	if c != nil {
		return c
	}

	// nacos todo get and listen
	// warm up conn
	return nil

}

var clusters map[string]*Cluster = make(map[string]*Cluster)
