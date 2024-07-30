package cluster

import "test/connPool"

type LbStrategy interface {
	Choose() (*Instance, error)
}

type Instance struct {
	cluster Cluster
	ip      string
	port    int
	Pool    *connPool.ConnPool
	tags    *map[string]string
}

type Cluster struct {
	name      string
	instances *[]Instance
	Lb        LbStrategy
}

func FindByName(name string) (*Cluster, error) {

	c := clusters[name]
	if c != nil {
		return c, nil
	}
	// nacos todo
	return nil, nil

}

var clusters map[string]*Cluster = make(map[string]*Cluster)
