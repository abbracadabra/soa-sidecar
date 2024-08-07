package shared

import (
	"errors"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// 不用返还，连接不是独占
// 独占式的连接用lru扩缩容
// 共享式的连接用qps扩缩容

func NewPool(size int, factory func() (*grpc.ClientConn, error)) *Pool {
	p := &Pool{
		factory: factory,
		conns:   make([]*PoolConn, size),
		locks:   make([]sync.Mutex, size),
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
	DEAD    State = iota
	HEALTHY       // 健康
)

type PoolConn struct {
	state       State
	idx         int // 在array中的位置
	pool        *Pool
	conn        *grpc.ClientConn
	activeLease int //缩容时kill

	//如果每个conn都有少量的qps lease,那就永远缩容不了了  。状态
}

func (cp *PoolConn) Unhealthy() {
	// cp.pool.Lock()
	cp.pool.locks[cp.idx].Lock()
	defer cp.pool.locks[cp.idx].Unlock()
	// defer cp.pool.Unlock()
	if cp.state == DEAD {
		return
	}
	cp.conn.Close()
	cp.state = DEAD
	newPc := &PoolConn{
		state: DEAD,
		idx:   cp.idx,
		pool:  cp.pool,
		conn:  cp.conn,
	}
	cp.pool.conns[cp.idx] = newPc
}

func (cp *PoolConn) Return() {
}

type Pool struct {
	sync.Mutex
	factory func() (*grpc.ClientConn, error)
	conns   []*PoolConn
	pos     int // roundrobin 位置
	// targetQpsPerConn int // 用于连接池的扩缩容
	// currentQps       int32
	locks []sync.Mutex
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
		return nil, dead
	}()
	if c1 != nil {
		// atomic.AddInt32(&p.currentQps, 1)
		return c1, nil
	}
	if c2 != nil {
		var err error
		p.locks[c2.idx].Lock()
		defer p.locks[c2.idx].Unlock()
		if c2.state == HEALTHY {
			return c2, nil
		}
		c2.conn, err = p.factory()
		if err != nil {
			return nil, err
		}
		c2.state = HEALTHY
		return c2, nil
	}
	return nil, errors.New("no conn")
}
