package ttls

import (
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"test/cluster"
	"test/localInstance"
	"test/utils/servNameUtil"
)

func ServeListenerIn(ln net.Listener, servName string, ins *localInstance.LocalInstance) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s", err)
			continue
		}

		tlsConn := conn.(*tls.Conn)
		err = tlsConn.Handshake()
		if err != nil {
			continue
		}
		state := tlsConn.ConnectionState()
		go createHandler(&proxyPhaseHookIn{ins: ins}, state.ServerName)(conn)
	}
}

func createHandler(hook proxyPhaseHook, servName string) func(conn net.Conn) {
	return func(conn net.Conn) {
		defer conn.Close()

		dstConn, err := hook.director(servName)
		if err != nil {
			return
		}
		defer dstConn.Close()
		sig1 := make(chan error)
		sig2 := make(chan error)
		go func() {
			_, err := io.Copy(dstConn, conn)
			sig1 <- err
		}()
		go func() {
			_, err := io.Copy(conn, dstConn)
			sig2 <- err
		}()
		<-sig1
		<-sig2
	}
}

func ServeListenerOut(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s", err)
			continue
		}

		tlsConn := conn.(*tls.Conn)
		err = tlsConn.Handshake()
		if err != nil {
			continue
		}
		state := tlsConn.ConnectionState()
		go createHandler(&proxyPhaseHookOut{}, state.ServerName)(conn)
	}
}

type proxyPhaseHook interface {
	director(serverName string) (net.Conn, error)
}

type proxyPhaseHookIn struct {
	proxyPhaseHook
	ins *localInstance.LocalInstance
}

func (c *proxyPhaseHookIn) director(serverName string) (net.Conn, error) {
	return net.Dial("tcp", c.ins.Ip+":"+strconv.Itoa(c.ins.Port))
}

type proxyPhaseHookOut struct {
	proxyPhaseHook
}

func (*proxyPhaseHookOut) director(host string) (net.Conn, error) {
	servName := servNameUtil.ExtractServName(host)
	cls := cluster.GetOrCreate(servName) //集群
	// tls只能访问主泳道，tls他没header
	ins := cls.Choose(&cluster.RouteInfo{Color: "main"}) //实例
	if ins == nil {
		return nil, errors.New("no instance")
	}
	return net.Dial("tcp", ins.IP+":"+strconv.Itoa(ins.Port))
}
