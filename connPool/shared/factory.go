package shared

import "time"

// 从左到右，重复使用一个健康的连接，直至连接的并发请求上限，若所有的连接达到并发上限，则恢复并使用unhealthy连接，若unhealthy连接都没有则用达到并发上限的连接
func NewPool[T any](size, initSize int, maxConcurrentStream int32, maxIdleTime time.Duration, factory func() (T, error), closeFunc func(T)) *Pool[T] {
	// 连接的并发数>maxConcurrentStream则用其他连接，并发数=0时销毁连接
	if size <= 0 {
		panic("size must be greater than 0")
	}
	if maxConcurrentStream <= 0 {
		panic("maxConcurrentStream must be greater than 0")
	}
	if factory == nil {
		panic("factory must be not nil")
	}
	p := &Pool[T]{
		factory: factory,
		conns:   make([]*PoolConn[T], size),
		// locks:               make([]sync.Mutex, size),
		maxConcurrentStream: maxConcurrentStream,
		onConnClose:         closeFunc,
	}

	for i, _ := range p.conns {
		p.conns[i] = &PoolConn[T]{
			healthy: false,
			idx:     i,
			pool:    p,
		}
	}
	go func() {
		for i := 0; i < initSize; i++ {
			func() {
				c := p.conns[i]
				c.Lock()
				defer c.Unlock()
				if c.healthy {
					return
				}
				newConn, err := factory()
				if err != nil {
					return
				}
				newPc := &PoolConn[T]{
					healthy: true,
					idx:     c.idx,
					pool:    c.pool,
					Conn:    newConn,
				}
				p.conns[i] = newPc
			}()
		}
	}()
	go func() {
		ticker := time.NewTicker(maxIdleTime)
		defer ticker.Stop()

		for range ticker.C {
			var close []interface{}
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
				p.onConnClose(gc)
			}
		}
	}()
	return p
}
