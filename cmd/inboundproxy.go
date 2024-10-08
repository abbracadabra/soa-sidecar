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

var inTransInServRule = make(map[string][]any)
var inTransLnMap = make(map[string]*helper.ChanListener)
var inTransOnceMap = sync.Map{}

func startTransparentIn(ip string, port int) error {
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
		dstPort := 0 // todo https://gist.github.com/fangdingjun/11e5d63abe9284dc0255a574a76bbcb1
		spec, ok := inTransInServRule[dstIp+":"+strconv.Itoa(dstPort)]
		if !ok {
			return nil
		}
		_once, _ := inTransOnceMap.LoadOrStore(strconv.Itoa(port), &sync.Once{})
		once := _once.(*sync.Once)
		once.Do(func() {
			secure := spec[0].(bool)
			protocol := spec[1].(string)
			ins := spec[2].(*localInstance.LocalInstance)
			servName := spec[3].(string)
			lis := helper.NewChanListener(dstIp, dstPort)
			inTransLnMap[strconv.Itoa(port)] = lis
			routeIn(lis, servName, ins, secure, protocol)
		})
		inTransLnMap[strconv.Itoa(port)].Supply(conn)
	}
}

func serveProtocolIn(servName string, ip string, port int, proxyIp string, proxyPort int, transparent bool, secure bool, protocol string) error {

	var ins = &localInstance.LocalInstance{
		ServName: servName,
		Ip:       ip,
		Port:     port,
	}

	var pf func(*localInstance.LocalInstance) (interface{}, error)
	if protocol == "grpc" {
		pf = grpcc.PoolFactoryIn
	}
	_pool, err := pf(ins)
	if err != nil {
		return err
	}
	ins.Pool = _pool

	if transparent {
		inTransInServRule[proxyIp+":"+strconv.Itoa(proxyPort)] = []any{secure, protocol, ins, servName}
		return nil
	}

	ln, err := net.Listen("tcp", proxyIp+":"+strconv.Itoa(proxyPort))
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return nil
	}

	return routeIn(ln, servName, nil, secure, protocol)
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
		httpp.ServeListenerIn(ln, servName, ins)
		return nil
	}
	if protocol == "grpc" {
		grpcc.ServeListenerIn(ln, servName, ins)
		return nil
	}

	if secure {
		ttls.ServeListenerIn(ln, servName, ins)
		return nil
	}

	return errors.New("unknown protocol")
}
