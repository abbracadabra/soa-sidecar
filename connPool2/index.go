package connPool

import (
	"errors"
	"net"
	"sync"
	"time"
	// cmap "github.com/orcaman/concurrent-map/v2"
)

type PoolConn struct {
	unhealthy   bool
	Conn        net.Conn
	lastPutTime time.Time
}

type ConnPool struct {
	freeConns       chan *PoolConn
	maxConns        int
	factory         func() (net.Conn, error)
	maxFreeDuration time.Duration
}

func NewConnPool(active, maxConns int, maxFreeDuration, waitDuration time.Duration, factory func() (net.Conn, error)) *ConnPool {
	pool := &ConnPool{
		freeConns:       make(chan *PoolConn, maxConns),
		maxConns:        maxConns,
		factory:         factory,
		maxFreeDuration: maxFreeDuration,
	}
	go pool.cleaner() // 启动清理空闲连接的
	return pool
}

func (p *ConnPool) cleaner() {
	ticker := time.NewTicker(p.maxFreeDuration / 2)
	defer ticker.Stop()

	for range ticker.C {
		for pc := range p.freeConns {
			if time.Since(pc.lastPutTime) > p.maxFreeDuration {
				pc.Conn.Close()
			} else {
				p.giveBack(pc.Conn, false)
			}
		}
	}
}

func (p *ConnPool) Get(waitTime time.Duration) (net.Conn, func(bool), error) {
	select {
	case pc := <-p.freeConns:
		var once sync.Once
		return pc.Conn, func(broken bool) {
			//防止多次调用
			once.Do(func() {
				p.giveBack(pc.Conn, broken)
			})
		}, nil
	case <-time.After(waitTime):
		return nil, nil, errors.New("get conn time out")
	}
}

// broken：连接坏了不能再用
func (p *ConnPool) giveBack(conn net.Conn, broken bool) {

	if broken {
		conn.Close()
		newConn, _ := p.factory()
		if newConn == nil {
			return
		}
		conn = newConn
	}
	pc := &PoolConn{
		Conn:        conn,
		lastPutTime: time.Now(),
	}
	select {
	case p.freeConns <- pc:
	default:
		pc.Conn.Close()
	}
}
