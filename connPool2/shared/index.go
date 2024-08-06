package shared

import (
	"errors"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// 不用返还，连接不是独占

func NewPool(size int, factory func() (*grpc.ClientConn, error)) *Pool {
	p := &Pool{
		factory: factory,
		conns:   make([]*PoolConn, size),
	}
	for i, _ := range p.conns {
		c, err := factory()
		if err != nil {
			p.conns[i] = &PoolConn{
				state: DEAD,
				idx:   i,
				pool:  p,
			}
			continue
		}
		p.conns[i] = &PoolConn{
			state: HEALTHY,
			idx:   i,
			pool:  p,
			conn:  c,
		}
	}
	return p
}

type State int

const (
	DEAD State = iota
	CONNECTING
	HEALTHY
)

type PoolConn struct {
	state State
	idx   int
	pool  *Pool
	conn  *grpc.ClientConn
}

func (cp *PoolConn) Unhealthy() {

	err := cp.conn.Close()

	cp.pool.Lock()
	defer cp.pool.Unlock()
	if cp.state == DEAD {
		return
	}
	cp.state = DEAD
	newPc := &PoolConn{
		state: DEAD,
		idx:   cp.idx,
		pool:  cp.pool,
	}
	cp.pool.conns[cp.idx] = newPc

}

type Pool struct {
	sync.Mutex
	factory func() (*grpc.ClientConn, error)
	conns   []*PoolConn
	pos     int
}

// todo 是否全搞成ctx
func (p *Pool) Get(waitTime time.Duration) (interface{}, error) {

	c1, c2 := func() (*PoolConn, *PoolConn) {
		p.Lock()
		defer p.Unlock()
		var dead *PoolConn
		for i := 0; i < len(p.conns); i++ {
			pc := p.conns[(p.pos+i)%len(p.conns)]
			if pc.state == HEALTHY {
				return pc, nil
			}
			if pc.state == DEAD {
				dead = pc
			}
		}
		p.pos = (p.pos + 1) % len(p.conns)
		if dead != nil {
			dead.state = CONNECTING
		}
		return nil, dead
	}()
	if c1 != nil {
		return c1, nil
	}
	if c2 != nil {
		var err error
		defer func() {
			if c2.state == CONNECTING {
				c2.state = DEAD
			}
		}()
		c2.conn, err = p.factory()
		if err != nil {
			c2.state = DEAD
			return nil, err
		}
		c2.state = HEALTHY
		return c2, nil
	}
	return nil, errors.New("no conn")
}
