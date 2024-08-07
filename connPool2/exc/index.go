package exc

import (
	"errors"
	"net"
	"sync"
	"time"
)

//todo 用完后放在头部、
//双chan activechan 和 deadlist
//timeout active close

// https://github.com/processout/grpc-go-pool/blob/master/pool.go

type PoolConn struct {
	sync.Mutex
	Conn        net.Conn
	pool        *Pool
	healthy     bool
	lastPutTime time.Time
}

type Pool struct {
	freeConns       chan *PoolConn
	deadConns       chan *PoolConn
	factory         func() (net.Conn, error)
	maxFreeDuration time.Duration
}

func NewConnPool(init, maxConns int, maxFreeDuration, waitDuration time.Duration, factory func() (net.Conn, error)) *Pool {
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
		freeConns:       make(chan *PoolConn, maxConns),
		factory:         factory,
		maxFreeDuration: maxFreeDuration,
	}

	go func() {
		for i := 0; i < init; i++ {
			c, err := factory()
			if err != nil {
				pool.deadConns <- &PoolConn{
					pool: pool,
				}
			} else {
				pool.freeConns <- &PoolConn{
					healthy:     true,
					Conn:        c,
					pool:        pool,
					lastPutTime: time.Now(),
				}
			}
		}
		for i := 0; i < maxConns-init; i++ {
			pool.freeConns <- &PoolConn{
				pool: pool,
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(pool.maxFreeDuration / 3)
		defer ticker.Stop()

		for range ticker.C {
		Loop:
			for {
				select {
				case pc := <-pool.freeConns:
					if time.Since(pc.lastPutTime) > maxFreeDuration {
						pc.Conn.Close()
						pc.healthy = false
						pool.deadConns <- pc
					} else {
						pool.freeConns <- pc
					}
				default:
					break Loop
				}
			}
		}
	}()

	return pool
}

func (p *Pool) Get(waitTime time.Duration) (interface{}, error) {
	select {
	case pc := <-p.freeConns:
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

func (p *PoolConn) Return() {
	if !p.healthy {
		p.Conn.Close()
	}
	p.Lock()
	defer p.Unlock()
	if p.Conn == nil {
		return
	}
	wrap := &PoolConn{
		Conn:        p.Conn,
		pool:        p.pool,
		lastPutTime: time.Now(),
	}
	if wrap.healthy {
		wrap.pool.freeConns <- wrap
	} else {
		wrap.pool.deadConns <- wrap
	}
	p.Conn = nil
}
