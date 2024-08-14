package ttls

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"test/grpcc"
	"test/httpp"
)

func startTls(protocol string, port int) {

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return
	}
	defer ln.Close()

	tlsListener := tls.NewListener(ln, &tls.Config{
		GetCertificate: GetCertificateForSNI,
		MinVersion:     tls.VersionTLS12,
	})

	defer tlsListener.Close()

	if protocol == "http" {
		httpp.ServeConnListener(tlsListener) // 因为grpc的tls alpn是http2，这里还不能知道是不是grpc
	}
	if protocol == "grpc" {
		grpcc.ServeConnListener(tlsListener)
	}
}

func startPlain(protocol string, port int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return
	}
	defer ln.Close()

	if protocol == "http" {
		httpp.ServeConnListener(ln) // 因为grpc的tls alpn是http2，这里还不能知道是不是grpc
	}
	if protocol == "grpc" {
		grpcc.ServeConnListener(ln)
	}
}
