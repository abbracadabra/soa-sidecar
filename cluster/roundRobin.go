package cluster

import "sync"

type RoundRobin struct {
	cls     *Cluster
	current int
	mu      sync.Mutex
}

func (r *RoundRobin) Choose() *Instance {
	r.mu.Lock()
	defer r.mu.Unlock()
	w := r.cls.instances[r.current]
	r.current = (r.current + 1) % len(r.cls.instances)
	return w
}

func NewRoundRobin(cluster *Cluster) *RoundRobin {
	return &RoundRobin{
		cls:     cluster,
		current: 0,
	}
}
