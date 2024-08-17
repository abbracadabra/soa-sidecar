package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"test/grpcc"
	"test/httpp"
	"test/ttls"
)

func main() {
	// startServe(false, "grpc", 8110, true)
}

var servRule = make(map[string][]any)

func StartTransparent(port int, outbound bool) error {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
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
		dstPort := 0
		spec := servRule[strconv.Itoa(dstPort)]
		secure := spec[0].(bool)
		protocol := spec[1].(string)
		route(conn, secure, protocol)
	}
}

func AddTransRule(port int, secure bool, protocol string) {
	servRule[strconv.Itoa(port)] = []any{secure, protocol}
}

func ServeProtocol(port int, outbound bool, secure bool, protocol string) error {

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
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

		route(conn, secure, protocol)
	}
}

func route(conn net.Conn, secure bool, protocol string) error {

	if secure {
		// Most uses of this package need not call Handshake explicitly: the first Conn.Read or Conn.Write will call it automatically.
		conn = tls.Server(conn, &tls.Config{
			GetCertificate: ttls.GetCertificateForSNI,
			MinVersion:     tls.VersionTLS12,
		})
		// ln = tls.NewListener(ln)

		// defer ln.Close()
	}

	if protocol == "http" {
		httpp.Channel <- conn
		// httpp.ServeConnListener(ln)
		return nil
	}
	if protocol == "grpc" {
		grpcc.Channel <- conn
		// grpcc.ServeConnListener(ln)
		return nil
	}

	if secure {
		ttls.Channel <- conn
		// ttls.ServeConnListener(ln)
		return nil
	}

	return errors.New("unknown protocol")
}
