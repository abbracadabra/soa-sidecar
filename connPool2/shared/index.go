package shared

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
)

// 不用返还，连接不是独占
// 独占式的连接用lru扩缩容
// 共享式的连接用qps扩缩容
// factory:返回连接、关闭方法、错误
func NewPool(size, initSize int, maxConcurrentStream int32, maxIdleTime time.Duration, factory func() (*grpc.ClientConn, func(), error)) *Pool {
	if size <= 0 {
		panic("size must be greater than 0")
	}
	if maxConcurrentStream <= 0 {
		panic("maxConcurrentStream must be greater than 0")
	}
	if factory == nil {
		panic("factory must be not nil")
	}
	p := &Pool{
		factory:             factory,
		conns:               make([]*PoolConn, size),
		locks:               make([]sync.Mutex, size),
		maxConcurrentStream: maxConcurrentStream,
	}

	for i, _ := range p.conns {
		p.conns[i] = &PoolConn{
			healthy: false,
			idx:     i,
			pool:    p,
		}
	}
	go func() {
		for i := 0; i < initSize; i++ {
			func() {
				p.locks[i].Lock()
				defer p.locks[i].Unlock()
				c := p.conns[i]
				if c.healthy {
					return
				}
				var err error
				c.Conn, c.closeFunc, err = factory()
				if err == nil {
					c.healthy = true
				}
			}()
		}
	}()
	go func() {
		ticker := time.NewTicker(maxIdleTime)
		defer ticker.Stop()

		for range ticker.C {
			var close []*grpc.ClientConn
			func() {
				p.Lock()
				defer p.Unlock()
				for _, c := range p.conns {
					if c.concurrentStream <= 0 && c.healthy {
						c.healthy = false
						close = append(close, c.Conn)
					}
				}
			}()
			for _, gc := range close {
				gc.Close()
			}
		}
	}()
	return p
}

type Lease struct {
	healthy bool
	conn    *PoolConn
}

func (l *Lease) Unhealthy() {
	l.healthy = false
}

func (l *Lease) Return() {
	l.conn.onReturn(l.healthy)
}

type PoolConn struct {
	sync.Mutex
	healthy          bool
	idx              int // 在array中的位置
	pool             *Pool
	Conn             *grpc.ClientConn
	concurrentStream int32
	closeFunc        func()
}

func (pc *PoolConn) onReturn(healthy bool) {
	defer atomic.AddInt32(&pc.concurrentStream, -1)
	pc.healthy = healthy
	if !healthy {
		pc.closeFunc()
	}
}

// func (pc *PoolConn) Unhealthy() {
// 	pc.healthy = false
// }

// func (pc *PoolConn) Return() {

// 	pc.pool.locks[pc.idx].Lock()
// 	defer pc.pool.locks[pc.idx].Unlock()
// 	// if pc.pool.conns[pc.idx] != pc {
// 	// 	//done
// 	// 	return
// 	// }
// 	// newPc := &PoolConn{
// 	// 	healthy: pc.healthy,
// 	// 	idx:     pc.idx,
// 	// 	pool:    pc.pool,
// 	// 	Conn:    pc.Conn,
// 	// }
// 	if !pc.healthy {
// 		pc.closeFunc()
// 	}
// 	atomic.AddInt32(&pc.concurrentStream, -1)
// 	// pc.pool.conns[pc.idx] = newPc
// }

type Pool struct {
	sync.Mutex
	factory             func() (*grpc.ClientConn, func(), error)
	conns               []*PoolConn
	locks               []sync.Mutex
	maxConcurrentStream int32
}

func (p *Pool) Shutdown() {
	p.Lock()
	defer p.Unlock()
	for _, pc := range p.conns {
		pc.closeFunc()
	}
}
func (p *Pool) Get(waitTime time.Duration) (*Lease, error) {
	var conn *PoolConn
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
func (p *Pool) tryGet() (*PoolConn, error) {

	c1, c2, c3 := func() (*PoolConn, *PoolConn, *PoolConn) {
		p.Lock()
		defer p.Unlock()
		var dead *PoolConn
		var defaultt *PoolConn
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
		atomic.AddInt32(&c1.concurrentStream, 1)
		return c1, nil
	}
	if c2 != nil {
		var err error
		p.locks[c2.idx].Lock()
		defer p.locks[c2.idx].Unlock()
		if c2.healthy {
			atomic.AddInt32(&c2.concurrentStream, 1)
			return c2, nil
		}
		c2.Conn, c2.closeFunc, err = p.factory()
		if err != nil {
			return nil, err
		}
		c2.healthy = true
		atomic.AddInt32(&c2.concurrentStream, 1)
		return c2, nil
	}
	if c3 != nil {
		atomic.AddInt32(&c3.concurrentStream, 1)
		return c3, nil
	}

	return nil, errors.New("no conn left")
}
