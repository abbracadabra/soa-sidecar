package ttls

import (
	"crypto/tls"
	"net"
)

var Channel = make(chan net.Conn)

func ServeConnListener() {
	for {
		select {
		case conn := <-Channel:
			tlsConn := conn.(*tls.Conn)
			err := tlsConn.Handshake()
			if err != nil {
				continue
			}

			state := tlsConn.ConnectionState()

			handleConn := myHandler{
				ConnectionState: state,
			}
			go handleConn.handleConn(conn)
		}
	}
}

type myHandler struct {
	tls.ConnectionState
}

func (mh *myHandler) handleConn(conn net.Conn) {

}

// for {
// 	conn, err := ln.Accept()
// 	if err != nil {
// 		continue
// 	}
// 	tlsConn := conn.(*tls.Conn)
// 	err = tlsConn.Handshake()
// 	if err != nil {
// 		continue
// 	}

// 	state := tlsConn.ConnectionState()

// 	handleConn := myHandler{
// 		ConnectionState: state,
// 	}
// 	go handleConn.handleConn(conn)
// }
