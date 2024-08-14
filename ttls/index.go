package ttls

import (
	"crypto/tls"
	"fmt"
	"net"
)

func start() {

	ln, err := net.Listen("tcp", ":8110")
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return
	}
	defer ln.Close()

	tlsListener := tls.NewListener(ln, &tls.Config{
		GetCertificate: getCertificateForSNI,
		MinVersion:     tls.VersionTLS12,
	})
}

func getCertificateForSNI(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	fmt.Println("start servingggg cert xokk " + clientHello.ServerName)
	domain := clientHello.ServerName
	cert, err := GetCert(domain)
	if err != nil {
		return nil, err
	}
	fmt.Println("start servingggg cert okk")
	return cert, nil
}
