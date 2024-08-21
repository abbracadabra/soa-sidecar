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
	"test/localInstance"
	"test/ttls"
)

var servRule = make(map[string][]any)
var transLnMap = make(map[string]*helper.ChanListener)
var onceMap = sync.Map{}

func startTransparent(ip string, port int, outbound bool) error {
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
		dstPort := 0 // todo 获取原始dst
		spec, ok := servRule[strconv.Itoa(dstPort)]
		if !ok {
			return nil
		}
		_once, _ := onceMap.LoadOrStore(strconv.Itoa(port), &sync.Once{})
		once := _once.(*sync.Once)
		once.Do(func() {
			secure := spec[0].(bool)
			protocol := spec[1].(string)
			lis := helper.NewChanListener(dstIp, dstPort)
			transLnMap[strconv.Itoa(port)] = lis
			if outbound {
				routeOut(lis, secure, protocol)
			} else {
				routeIn(lis)
			}
		})
		transLnMap[strconv.Itoa(port)].Supply(conn)
	}
}

func serveProtocolOut(transparent bool, ip string, port int, secure bool, protocol string) error {

	if transparent {
		servRule[strconv.Itoa(port)] = []any{secure, protocol}
	}

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return nil
	}

	return routeOut(ln, secure, protocol)
}

func serveProtocolIn(servName string, transparent bool, ip string, port int, secure bool, protocol string) error {

	localInstance.AddInstance(ip, port, xx)

	if transparent {
		servRule[strconv.Itoa(port)] = []any{secure, protocol}
	}

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return nil
	}

	return routeIn(ln, servName, nil, secure, protocol)
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
		httpp.ServeConnListener(ln, true)
		return nil
	}
	if protocol == "grpc" {
		grpcc.ServeConnListener(ln, true)
		return nil
	}

	if secure {
		ttls.ServeConnListener(ln, true)
		return nil
	}

	return errors.New("unknown protocol")
}
func routeIn(ln net.Listener, servName string, ins *localInstance.LocalInstance, secure bool, protocol string) error {

	if secure {
		// Most uses of this package need not call Handshake explicitly: the first Conn.Read or Conn.Write will call it automatically.
		ln = tls.NewListener(ln, &tls.Config{
			GetCertificate: ttls.GetCertificateForSNI,
			MinVersion:     tls.VersionTLS12,
		})
	}

	if protocol == "http" {
		httpp.ServeConnListener(ln, servName, ins)
		return nil
	}
	if protocol == "grpc" {
		grpcc.ServeConnListener(ln, servName, ins)
		return nil
	}

	if secure {
		ttls.ServeConnListener(ln, servName, ins)
		return nil
	}

	return errors.New("unknown protocol")
}
