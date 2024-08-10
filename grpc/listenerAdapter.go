package main

import (
	"net"
	"sync"
)

// SingleConnListener 是一个包装了单个 net.Conn 的 Listener，只能接受一次连接。
type SingleConnListener struct {
	conn      net.Conn
	accepted  bool
	acceptMux sync.Mutex
}

// Accept 返回包装的 net.Conn，只能接受一次。
func (l *SingleConnListener) Accept() (net.Conn, error) {
	l.acceptMux.Lock()
	defer l.acceptMux.Unlock()
	if l.accepted {
		return nil, &net.OpError{
			Op:   "accept",
			Net:  "tcp",
			Addr: l.conn.LocalAddr(),
			Err:  net.ErrClosed,
		}
	}
	l.accepted = true
	return l.conn, nil
}

// Close 关闭包装的连接。
func (l *SingleConnListener) Close() error {
	// return l.conn.Close() 别关，不然client会报conn reset
	return nil
}

// Addr 返回包装连接的本地地址。
func (l *SingleConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

// NewSingleConnListener 包装一个 net.Conn 成为一个 SingleConnListener。
func NewSingleConnListener(conn net.Conn) net.Listener {
	return &SingleConnListener{
		conn: conn,
	}
}

// func main() {
// 	// 示例使用
// 	listener := NewSingleConnListener(someConn) // someConn 是你要包装的 net.Conn
// 	conn, err := listener.Accept()
// 	if err != nil {
// 		// 处理错误
// 	}
// 	defer conn.Close()

// 	// 使用 conn 进行读写
// }
