package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/http2"
)

func main() {
	addr := ":8087"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}
	defer listener.Close()

	log.Printf("Listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	var h2s = &http2.Server{}
	h2s.ServeConn(conn, &http2.ServeConnOpts{
		Handler: http.HandlerFunc(handleServerInbound),
	})

}

func handleServerInbound(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// optional transcoding

	tr := &http2.Transport{
		AllowHTTP: true, // 允许不使用TLS的HTTP/2连接
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr) // 使用net.Dial进行明文连接
		},
	}

	client := &http.Client{
		Transport: tr,
	}

	req, err := http.NewRequest(r.Method, getFullURL(r, "127.0.0.1:8010"), r.Body)
	if err != nil {
		panic(err)
	}
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("cli error:" + err.Error())
		if req.Body != nil {
			_, _ = io.Copy(io.Discard, req.Body)
			req.Body.Close()
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	isDone := make(chan bool)
	go handleClientInbound(w, resp, err, isDone)
	<-isDone // 据说不等就结束方法的话，response会关闭，所以等一下
}

func handleClientInbound(w http.ResponseWriter, resp *http.Response, err error, isDone chan bool) {
	defer func() {
		isDone <- true
	}()
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// optional transcoding

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func getFullURL(req *http.Request, host string) string {
	// 使用url.URL结构体来构建完整的URL
	u := &url.URL{
		Scheme:   req.URL.Scheme,
		Host:     host,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
		Fragment: req.URL.Fragment,
	}

	// 如果原始请求的URL没有包含Scheme，可以从Request对象中获取
	if u.Scheme == "" {
		// if req.TLS != nil {
		// 	u.Scheme = "https"
		// } else {
		u.Scheme = "http"
		// }
	}

	return u.String()
}
