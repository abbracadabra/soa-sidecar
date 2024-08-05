package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"test/cluster"
)

func main() {
	ln, err := net.Listen("tcp", ":8086")
	if err != nil {
		fmt.Println("Error setting up TCP listener:", err)
		return
	}
	defer ln.Close()

	fmt.Println("Listening on :8083")

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			fmt.Println("Error reading request:", err)
			return
		}

		handleServerInbound(req, conn)
	}

}

func handleServerInbound(r *http.Request, conn net.Conn) {

	defer r.Body.Close()
	// optional transcoding
	cls := cluster.FindByName("example")
	ins := cls.Lb.Choose()

	addr := ins.IP + ":" + strconv.Itoa(ins.Port)
	req, err := http.NewRequest(r.Method, getFullURL(r, addr), r.Body)
	req.Host = r.Host
	if err != nil {
		panic(err)
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	//todo write to tcp conn
	client := &http.Client{}
	resp, err := client.Do(req) // 这里报错，r.Body会读完吗
	if err != nil {
		fmt.Println("cli error:" + err.Error())
		if req.Body != nil {
			_, _ = io.Copy(io.Discard, req.Body)
			req.Body.Close()
		}
		_, err := conn.Write([]byte(errResponse))
		return
	}

	defer resp.Body.Close()
	isDone := make(chan bool)
	go handleClientInbound(conn, resp, err, isDone)
	<-isDone // http1 等响应后才能处理下个请求
}

func handleClientInbound(conn net.Conn, resp *http.Response, err error, isDone chan bool) {
	defer func() {
		isDone <- true
	}()
	if err != nil {
		return
	}

	defer resp.Body.Close()

	// optional transcoding

	conn.Write([]byte(resp.Proto + " " + resp.Status + "\r\n"))
	resp.Header.Write(conn)
	conn.Write([]byte("\r\n"))
	io.Copy(conn, resp.Body)
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
		if req.TLS != nil {
			u.Scheme = "https"
		} else {
			u.Scheme = "http"
		}
	}

	return u.String()
}

var errResponse = "HTTP/1.1 500 Internal Server Error\r\n" +
	"Content-Type: text/plain\r\n" +
	"Content-Length: 11\r\n" +
	"\r\n" +
	"proxy error"
