package connPool

import (
	"errors"
	"net"
	"time"
	// cmap "github.com/orcaman/concurrent-map/v2"
)

// var ActiveReqs = make(map[net.Conn][]interface{})

type Req struct {
	Sig chan bool
	Res interface{}
}

type ConnInfo struct {
}

var connMap = make(map[net.Conn]ConnInfo)

type Connection struct {
	Conn        net.Conn
	lastPutTime time.Time
	ActiveReqs  map[string]*Req
}

type ConnPool struct {
	// mu              sync.Mutex
	freeConns       chan net.Conn
	active          int
	maxConns        int
	factory         func() (net.Conn, error)
	maxFreeDuration time.Duration
	waitDuration    time.Duration
}

func NewConnPool(active, maxConns int, maxFreeDuration, waitDuration time.Duration, factory func() (net.Conn, error)) *ConnPool {
	pool := &ConnPool{
		freeConns:       make(chan net.Conn, maxConns),
		maxConns:        maxConns,
		factory:         factory,
		maxFreeDuration: maxFreeDuration,
		waitDuration:    waitDuration,
	}
	go pool.cleaner() // 启动清理空闲连接的 goroutine
	return pool
}

func (p *ConnPool) cleaner() {
	ticker := time.NewTicker(p.maxFreeDuration / 2)
	defer ticker.Stop()

	for range ticker.C {
		for pc := range p.freeConns {
			if time.Since(pc.lastPutTime) > p.maxFreeDuration {
				pc.Close()
				p.active--
			} else {
				p.GiveBack(pc, false)
			}
		}
	}
}

func (p *ConnPool) Get() (net.Conn, error) {
	select {
	case pc := <-p.freeConns:
		p.active++
		return pc.conn, nil
	default:
		if len(p.freeConns)+p.active < p.maxConns {
			conn, _ := p.factory()
			if conn != nil {
				p.active++
				return conn, nil
			}
		}
		select {
		case pc := <-p.freeConns:
			p.active++
			return pc.conn, nil
		case <-time.After(p.waitDuration):
			return nil, errors.New("timed out waiting for connection")
		}
	}
}

// close：是否关闭连接
func (p *ConnPool) GiveBack(conn net.Conn, close bool) {

	defer func() {
		p.active--
	}()

	if len(p.freeConns)+p.active >= p.maxConns {
		conn.Close()
		return
	}
	if close {
		conn.Close()
		newConn, _ := p.factory()
		if newConn == nil {
			return
		}
		conn = newConn
	}
	pc := &Connection{
		conn:        conn,
		lastPutTime: time.Now(),
	}
	select {
	case p.freeConns <- pc:
	default:
		conn.Close()
	}
}
