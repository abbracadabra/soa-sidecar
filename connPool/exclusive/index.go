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

type PoolConn[T any] struct {
	sync.Mutex
	Conn        interface{}
	pool        *Pool[T]
	healthy     bool
	lastPutTime time.Time
}

type Pool[T any] struct {
	idleConns   chan *PoolConn[T] `idleConns + deadConns + 在用的conn = max conn size`
	deadConns   chan *PoolConn[T]
	factory     func() (interface{}, error)
	onConnClose func(interface{})
	// maxIdleTime time.Duration
}

func (p *Pool[T]) Get(waitTime time.Duration) (*PoolConn[T], error) {
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

func (p *PoolConn[T]) Unhealthy() {
	p.healthy = false
}

func (pc *PoolConn[T]) Return() {
	pc.Lock()
	defer pc.Unlock()
	if pc.Conn == nil {
		return
	}
	if !pc.healthy {
		pc.pool.onConnClose(pc.Conn)
	}
	wrap := &PoolConn[T]{
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
