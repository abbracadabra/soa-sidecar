package shared

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type Lease[T any] struct {
	healthy bool
	conn    *PoolConn[T]
}

func (l *Lease[T]) GetConn() interface{} {
	return l.conn.Conn
}

func (l *Lease[T]) Unhealthy() {
	l.healthy = false
}

func (l *Lease[T]) Return() {
	l.conn.onLeaseReturn(l.healthy)
}

type PoolConn[T any] struct {
	sync.Mutex
	healthy          bool
	idx              int // 在array中的位置
	pool             *Pool[T]
	Conn             interface{}
	concurrentStream int32
	// leaseBeginCb     func() // ??
	// leaseEndCb       func() // ??
}

func (pc *PoolConn[T]) acquireLease() *Lease[T] {
	atomic.AddInt32(&pc.concurrentStream, 1)
	l := &Lease[T]{conn: pc, healthy: true}
	// pc.leaseBeginCb()
	return l
}

func (pc *PoolConn[T]) onLeaseReturn(healthy bool) {
	// defer pc.leaseEndCb()
	defer atomic.AddInt32(&pc.concurrentStream, -1)
	pc.healthy = healthy
	if !healthy {
		pc.pool.onConnClose(pc.Conn)
	}
}

type Pool[T any] struct {
	sync.Mutex
	factory             func() (T, error)
	conns               []*PoolConn[T]
	maxConcurrentStream int32
	onConnClose         func(T)
}

func (p *Pool[T]) Close() error {
	p.Lock()
	defer p.Unlock()
	for _, pc := range p.conns {
		p.onConnClose(pc.Conn)
	}
	return nil
}
func (p *Pool[T]) Get(waitTime time.Duration) (*Lease[T], error) {
	var conn *Lease[T]
	var err error

	retries := 2          // Maximum number of retries
	delay := waitTime / 2 // Divide waitTime by the number of retries

	for i := 0; i <= retries; i++ {
		conn, err = p.tryGet()
		if err == nil {
			return conn, nil // Success, return the connection
		}

		// If there's an error and we've used up our retries, break out
		if i == retries {
			break
		}

		// Wait for the specified delay before retrying
		time.Sleep(delay)
	}

	// Return the last error encountered after retrying
	return nil, err
}

// todo 是否全搞成ctx
func (p *Pool[T]) tryGet() (*Lease[T], error) {
	c1, c2, c3 := func() (*PoolConn[T], *PoolConn[T], *PoolConn[T]) {
		p.Lock()
		defer p.Unlock()
		var dead *PoolConn[T]
		var defaultt *PoolConn[T]
		for i := 0; i < len(p.conns); i++ {
			pc := p.conns[i]
			if pc.healthy && pc.concurrentStream <= p.maxConcurrentStream {
				return pc, nil, nil
			}
			if !pc.healthy {
				if dead == nil {
					dead = pc
				}
			} else {
				defaultt = pc
			}
		}
		return nil, dead, defaultt
	}()
	if c1 != nil {
		return c1.acquireLease(), nil
	}
	if c2 != nil {
		c2.Lock()
		defer c2.Unlock()
		if c2.healthy {
			return c2.acquireLease(), nil
		}
		newConn, err := p.factory()
		if err != nil {
			return nil, err
		}
		newC2 := &PoolConn[T]{
			healthy: true,
			idx:     c2.idx,
			pool:    c2.pool,
			Conn:    newConn,
		}
		p.conns[c2.idx] = newC2
		return newC2.acquireLease(), nil
	}
	if c3 != nil {
		return c3.acquireLease(), nil
	}

	return nil, errors.New("no conn left")
}
