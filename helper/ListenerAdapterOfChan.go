package helper

import (
	"net"
	"sync"
)

// NewChanListener 创建并返回一个 ChanListener 实例
func NewChanListener(connChan chan net.Conn) *ChanListener {
	return &ChanListener{
		connChan: connChan,
		// close:    make(chan struct{}),
		addr: &mockAddr{},
	}
}

// ChanListener 是一个自定义的 net.Listener，它从一个 chan net.Conn 获取连接
type ChanListener struct {
	connChan chan net.Conn
	// close    chan struct{}
	addr net.Addr
	once sync.Once
}

// Accept 实现了 net.Listener 接口，从 chan net.Conn 获取下一个连接
func (l *ChanListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.connChan:
		return conn, nil
		// case <-l.close:
		// 	return nil, net.ErrClosed
	}
}

// Close 关闭 Listener
func (l *ChanListener) Close() error {
	l.once.Do(func() {
		close(l.close)
	})
	return nil
}

// Addr 返回 Listener 的网络地址
func (l *ChanListener) Addr() net.Addr {
	return l.addr
}

// mockAddr 是一个简单的 net.Addr 实现
type mockAddr struct {
	network string
	address string
}

func (a *mockAddr) Network() string {
	return a.network
}

func (a *mockAddr) String() string {
	return a.address
}
