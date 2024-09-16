package exclusive

import "time"

func NewConnPool(init, maxConns int, maxIdleTime, waitDuration time.Duration, factory func() (interface{}, error), closeFunc func(interface{})) *Pool {
	if maxConns <= 0 {
		maxConns = 1
	}
	if init < 0 {
		init = 0
	}
	if init > maxConns {
		init = maxConns
	}
	pool := &Pool{
		idleConns: make(chan *PoolConn, maxConns),
		deadConns: make(chan *PoolConn, maxConns),
		factory:   factory,
		closeFunc: closeFunc,
		// maxIdleTime: maxIdleTime,
	}

	go func() {
		for i := 0; i < init; i++ {
			c, err := factory()
			if err != nil {
				pool.deadConns <- &PoolConn{
					pool: pool,
				}
			} else {
				pool.idleConns <- &PoolConn{
					healthy:     true,
					Conn:        c,
					pool:        pool,
					lastPutTime: time.Now(),
				}
			}
		}
		for i := 0; i < maxConns-init; i++ {
			pool.idleConns <- &PoolConn{
				pool: pool,
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(maxIdleTime / 3)
		defer ticker.Stop()

		for range ticker.C {
		Loop:
			for {
				select {
				case pc := <-pool.idleConns:
					if time.Since(pc.lastPutTime) > maxIdleTime {
						closeFunc(pc.Conn)
						pc.healthy = false
						pool.deadConns <- pc
					} else {
						pool.idleConns <- pc
					}
				default:
					break Loop
				}
			}
		}
	}()

	return pool
}
