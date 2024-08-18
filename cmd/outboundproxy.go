package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"test/grpcc"
	"test/helper"
	"test/httpp"
	"test/ttls"
)

var outServRule = make(map[string][]any)
var outTransLnMap = make(map[string]*helper.ChanListener)
var outTransLnOnceMap = sync.Map{}

func startTransparentOut(ip string, port int) error {
	ln, err := net.Listen("tcp", ip+":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return nil
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		dstIp := ""
		dstPort := 0 // todo
		spec, ok := outServRule[strconv.Itoa(dstPort)]
		if !ok {
			return nil
		}
		_once, _ := outTransLnOnceMap.LoadOrStore(strconv.Itoa(port), &sync.Once{})
		once := _once.(*sync.Once)
		once.Do(func() {
			secure := spec[0].(bool)
			protocol := spec[1].(string)
			lis := helper.NewChanListener(dstIp, dstPort)
			outTransLnMap[strconv.Itoa(port)] = lis
			routeOut(lis, secure, protocol)
		})
		outTransLnMap[strconv.Itoa(port)].Supply(conn)
	}
}

func serveProtocolOut(transparent bool, ip string, port int, secure bool, protocol string) error {

	if transparent {
		outServRule[strconv.Itoa(port)] = []any{secure, protocol}
	}

	ln, err := net.Listen("tcp", ip+":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return nil
	}

	return routeOut(ln, secure, protocol)
}

func routeOut(ln net.Listener, secure bool, protocol string) error {
	if secure {
		// Most uses of this package need not call Handshake explicitly: the first Conn.Read or Conn.Write will call it automatically.
		ln = tls.NewListener(ln, &tls.Config{
			GetCertificate: ttls.GetCertificateForSNI,
			MinVersion:     tls.VersionTLS12,
		})
	}

	if protocol == "http" {
		httpp.ServeListenerOut(ln)
		return nil
	}
	if protocol == "grpc" {
		grpcc.ServeListenerOut(ln)
		return nil
	}

	if secure {
		ttls.ServeListenerOut(ln)
		return nil
	}

	return errors.New("unknown protocol")
}
