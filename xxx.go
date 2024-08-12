package demohttp

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
)

var certStore = map[string]*tls.Certificate{
	// 这里存储的是示例证书，实际使用时你需要根据域名生成或加载证书
	"example.com": &tls.Certificate{ /* 加载证书数据 */ },
}

func getCertificateForSNI(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	domain := clientHello.ServerName
	if cert, ok := certStore[domain]; ok {
		return cert, nil
	}
	return nil, fmt.Errorf("no certificate found for domain %s", domain)
}

func main() {
	// 创建反向代理
	proxy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 这里可以实现实际的反向代理逻辑
		http.Error(w, "Proxy not implemented", http.StatusNotImplemented)
	})

	// 配置 TLS
	tlsConfig := &tls.Config{
		GetCertificate: getCertificateForSNI,
		MinVersion:     tls.VersionTLS13,
	}
	server := &http.Server{
		Handler:   proxy,
		TLSConfig: tlsConfig,
	}

	// 启动服务器
	log.Println("Starting HTTPS server on :443")
	err := server.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
