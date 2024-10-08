package ttls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"sync"
	"time"
)

func GetCertificateForSNI(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	domain := clientHello.ServerName
	cert, err := GetCert(domain)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// https://gist.github.com/shivakar/cd52b5594d4912fbeb46
var mu sync.Mutex
var certMap = make(map[string]*tls.Certificate)

func GetCert(domain string) (*tls.Certificate, error) {
	cert := certMap[domain]
	if cert != nil {
		return cert, nil
	}
	mu.Lock()
	defer mu.Unlock()
	cert = certMap[domain]
	if cert != nil {
		return cert, nil
	}
	newCert, err := doGenX509KeyPair(domain)
	if err != nil {
		return nil, err
	}
	certMap[domain] = &newCert
	return &newCert, nil
}

func doGenX509KeyPair(domain string) (tls.Certificate, error) {
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(now.Unix()),
		Subject: pkix.Name{
			CommonName:         domain,
			Country:            []string{"USA"},
			Organization:       []string{domain},
			OrganizationalUnit: []string{"quickserve"},
		},
		NotBefore:             now,
		NotAfter:              now.AddDate(1, 0, 0),
		SubjectKeyId:          []byte{113, 117, 105, 99, 107, 115, 101, 114, 118, 101},
		BasicConstraintsValid: true,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	cert, err := x509.CreateCertificate(rand.Reader, template, template,
		priv.Public(), priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	var outCert tls.Certificate
	outCert.Certificate = append(outCert.Certificate, cert)
	outCert.PrivateKey = priv

	return outCert, nil
}
