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
		myHandler := myHandler{
			servName: state.ServerName,
			cycle:    &inboundCycle{ins: ins},
		}
		go myHandler.handle(conn)
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
		myHandler := myHandler{
			servName: state.ServerName,
			cycle:    &outboundCycle{},
		}
		go myHandler.handle(conn)
	}
}

type myHandler struct {
	servName string
	cycle    lifeCycle
}

func (h *myHandler) handle(conn net.Conn) {
	defer conn.Close()

	dstConn, err := h.cycle.director(h.servName)
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

type lifeCycle interface {
	director(serverName string) (net.Conn, error)
}

type inboundCycle struct {
	lifeCycle
	ins *localInstance.LocalInstance
}

func (c *inboundCycle) director(serverName string) (net.Conn, error) {
	return net.Dial("tcp", c.ins.Ip+":"+strconv.Itoa(c.ins.Port))
}

type outboundCycle struct {
	lifeCycle
}

func (*outboundCycle) director(serverName string) (net.Conn, error) {
	cls := cluster.GetOrCreate(serverName, cluster.InstanceRouteByLaneCreator) //集群
	// tls只能访问主泳道，tls他没header
	ins := cls.Choose("main") //实例
	if ins == nil {
		return nil, errors.New("no instance")
	}
	return net.Dial("tcp", ins.IP+":"+strconv.Itoa(ins.Port))
}
