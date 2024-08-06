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
	Healthy     bool
	Conn        net.Conn
	pool        *Pool
	lastPutTime time.Time
}

type Pool struct {
	freeConns       chan *PoolConn
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
				pool.freeConns <- &PoolConn{
					pool: pool,
				}
			} else {
				pool.freeConns <- &PoolConn{
					Healthy:     true,
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

	return pool
}

func (p *Pool) Get(waitTime time.Duration) (interface{}, error) {
	var pc *PoolConn
	select {
	case pc = <-p.freeConns:
		//good
	case <-time.After(waitTime):
		return nil, errors.New("get conn time out")
	}
	if pc.lastPutTime.Add(p.maxFreeDuration).Before(time.Now()) {
		pc.Healthy = false
	}
	if !pc.Healthy {
		var err error
		pc.Conn, err = p.factory()
		if err != nil {
			p.freeConns <- &PoolConn{
				pool: p,
			}
			return nil, err
		}
		pc.lastPutTime = time.Now()
		pc.Healthy = true
	}
	return pc, nil
}

func (p *PoolConn) GiveBack() {
	p.Lock()
	if p.Conn == nil {
		//已返还
		return
	}
	wrap := &PoolConn{
		Healthy:     p.Healthy,
		Conn:        p.Conn,
		pool:        p.pool,
		lastPutTime: time.Now(),
	}
	if !wrap.Healthy {
		wrap.Conn.Close()
	}

	select {
	case wrap.pool.freeConns <- wrap:
	default:
		if wrap.Conn != nil {
			wrap.Conn.Close()
		}
	}
	p.Conn = nil
	p.Unlock()
}
