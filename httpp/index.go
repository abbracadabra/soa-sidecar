package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"test/cluster"
	"time"

	"golang.org/x/net/http2"
)

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

func main() {
	ln, err := net.Listen("tcp", ":8110")
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return
	}
	defer ln.Close()
	log.Println("Starting HTTPS server on :8110")

	//tls,这里可以tls.serve conn并返回conn
	tlsListener := tls.NewListener(ln, &tls.Config{
		GetCertificate: getCertificateForSNI,
		MinVersion:     tls.VersionTLS12,
	})
	//http1
	server := &http.Server{
		Handler: http.HandlerFunc(handleInbound),
	}
	//add http2 support to http1 server，这样能服务http1和2，http2.Server只支持http2
	err = http2.ConfigureServer(server, &http2.Server{})
	if err != nil {
		log.Fatalf("Failed to configure HTTP/2 server: %v", err)
	}

	err = server.Serve(tlsListener)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}

}

var tr = &http.Transport{
	MaxIdleConns:        0,                // 最大空闲连接数
	MaxIdleConnsPerHost: 50,               // 每个主机的最大空闲连接数
	IdleConnTimeout:     60 * time.Second, // 空闲连接的超时时间
}

var client = &http.Client{
	Transport: tr,
}

var finished = errors.New("finished")

func handleInbound(w http.ResponseWriter, r *http.Request) {
	fmt.Println("start servingggg  serv")

	var fwc = make(chan error, 1)
	var bwc = make(chan []any, 1)
	defer func() {
		select {
		case e := <-fwc:
			defer r.Body.Close()
			if errors.Is(e, finished) {
				return
			}
			len, _ := strconv.Atoi(r.Header.Get("Content-Length"))
			if len > 0 || r.Header.Get("Content-Encoding") == "chunked" {
				_, _ = io.Copy(io.Discard, r.Body)
			}
			http.Error(w, e.Error(), http.StatusInternalServerError)
		}
		select {
		case e := <-bwc:
			if e == nil {
				return
			}
			rsp := e[0].(*http.Response)
			err := e[1].(error)
			defer rsp.Body.Close()
			if errors.Is(err, finished) {
				return
			}
			if rsp.Body != nil {
				_, _ = io.Copy(io.Discard, rsp.Body)
			}
		}
	}()

	defer close(fwc)
	defer close(bwc)
	host := r.Header.Get("Host")
	cls := cluster.FindByName(host, true) //集群
	ins := cls.Choose()                   //实例  todo by 勇道
	if ins == nil {
		fwc <- errors.New("no instance")
		return
	}

	req, err := http.NewRequest(r.Method, getFullURL(r, ins.IP+":"+strconv.Itoa(ins.Port)), r.Body)
	if err != nil {
		fwc <- err
		return
	}
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := forwardReq(req, client)
	if err != nil {
		fwc <- err
		return
	}
	fwc <- finished
	err = forwardResp(resp, w)
	if err != nil {
		bwc <- []any{resp, err}
		return
	}
	bwc <- []any{resp, finished}
}

func forwardReq(r *http.Request, client *http.Client) (*http.Response, error) {
	return client.Do(r)
}
func forwardResp(resp *http.Response, w http.ResponseWriter) error {
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, err := io.Copy(w, resp.Body)
	return err
}

func getFullURL(req *http.Request, host string) string {
	u := &url.URL{
		Scheme:   "http",
		Host:     host,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
		Fragment: req.URL.Fragment,
	}

	return u.String()
}
