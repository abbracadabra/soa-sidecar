package cluster

import "sync"

type RoundRobin struct {
	LbStrategy
	cluster *Cluster
	current int
	mu      sync.Mutex
}

func (r *RoundRobin) Choose() *Instance {
	r.mu.Lock()
	defer r.mu.Unlock()
	// 选择当前索引的worker
	w := (*r.cluster.instances)[r.current]
	r.current = (r.current + 1) % len(*r.cluster.instances)
	return &w
}

func NewRoundRobin(cluster *Cluster) *RoundRobin {
	return &RoundRobin{
		cluster: cluster,
		current: 0,
	}
}
