package exclusive

import (
	"errors"
	"sync"
	"time"
)

//todo 用完后放在头部、
//双chan activechan 和 deadlist
//timeout active close

// https://github.com/processout/grpc-go-pool/blob/master/pool.go

type PoolConn struct {
	sync.Mutex
	Conn        interface{}
	pool        *Pool
	healthy     bool
	lastPutTime time.Time
}

type Pool struct {
	idleConns chan *PoolConn
	deadConns chan *PoolConn
	factory   func() (interface{}, error)
	closeFunc func(interface{})
	// maxIdleTime time.Duration
}

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

func (p *Pool) Get(waitTime time.Duration) (*PoolConn, error) {
	select {
	case pc := <-p.idleConns:
		return pc, nil
	case dead := <-p.deadConns:
		{
			c, err := p.factory()
			if err != nil {
				p.deadConns <- dead
				return nil, err
			}
			dead.Conn = c
			dead.healthy = true
			return dead, nil
		}
	case <-time.After(waitTime):
		return nil, errors.New("get conn time out")
	}
}

func (p *PoolConn) Unhealthy() {
	p.healthy = false
}

func (pc *PoolConn) Return() {
	pc.Lock()
	defer pc.Unlock()
	if pc.Conn == nil {
		return
	}
	if !pc.healthy {
		pc.pool.closeFunc(pc.Conn)
	}
	wrap := &PoolConn{
		Conn:        pc.Conn,
		pool:        pc.pool,
		lastPutTime: time.Now(),
	}
	if wrap.healthy {
		wrap.pool.idleConns <- wrap
	} else {
		wrap.pool.deadConns <- wrap
	}
	pc.Conn = nil
}
