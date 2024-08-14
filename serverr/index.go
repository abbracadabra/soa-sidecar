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
	startServe(false, "grpc", 8110)
}

func startServe(secure bool, protocol string, port int) error {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return nil
	}
	defer ln.Close()

	if secure {
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

	return errors.New("unknown protocol")
}
