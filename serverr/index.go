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
	startServe(false, "grpc", 8110, true)
}

var transPrxRule = make(map[string][]any)

func startTransparent() error {
	//get origin dst
	ln, err := net.Listen("tcp", ":9999")
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
		var originPort int //todo
		spec := transPrxRule[strconv.Itoa(originPort)]

		secure := spec[0].(bool)
		protocol := spec[1].(string)

		// chan <- conn
	}
}

func startServe(secure bool, protocol string, port int, bind bool) error {

	if !bind {
		transPrxRule[strconv.Itoa(port)] = [2]any{secure, protocol}
		return nil
	}

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return nil
	}
	defer ln.Close()

	if secure {
		// Most uses of this package need not call Handshake explicitly: the first Conn.Read or Conn.Write will call it automatically.
		ln = tls.NewListener(ln, &tls.Config{
			GetCertificate: ttls.GetCertificateForSNI,
			MinVersion:     tls.VersionTLS12,
		})

		defer ln.Close()
	}

	if protocol == "http" {
		httpp.ServeConnListener(ln)
		return nil
	}
	if protocol == "grpc" {
		grpcc.ServeConnListener(ln)
		return nil
	}

	if secure {
		ttls.ServeConnListener(ln)
		return nil
	}

	return errors.New("unknown protocol")
}
