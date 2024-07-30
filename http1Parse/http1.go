package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"test/cluster"
	"test/connPool"
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
	// clusterName := "example"
	// getCluster(clusterName) && start reading when create new conn
	var cliConn net.Conn
	var cluster cluster.Cluster
	ins, err := cluster.Lb.Choose()
	ins.Pool.Get()

	req, err := http.NewRequest(r.Method, getFullURL(r, "127.0.0.1:12345"), r.Body)
	// req.Host = "www.example.com"
	if err != nil {
		panic(err)
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	activeQ := &connPool.Req{}
	cliConn.ActiveReqs["1"] = &connPool.Req{}
	defer delete(cliConn.ActiveReqs, "1")

	// 写reqline
	fmt.Fprintf(cliConn.Conn, "%s %s HTTP/1.1\r\n", req.Method, req.URL.RequestURI())
	// 写入请求头
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Fprintf(cliConn.Conn, "%s: %s\r\n", textproto.CanonicalMIMEHeaderKey(key), value)
		}
	}
	fmt.Fprintf(cliConn.Conn, "\r\n")
	//写body
	if req.Body != nil {
		defer req.Body.Close()
		_, err = io.Copy(conn, req.Body)
		if err != nil {
			fmt.Println("Error writing request body:", err)
			_, _ = io.Copy(io.Discard, req.Body)
			return
		}
	}
	<-activeQ.Sig
	res := activeQ.Res

	handleClientInbound(conn, res.(*http.Response), err)
}

func handleClientInbound(conn net.Conn, resp *http.Response, err error) {
	if err != nil {
		_, err := conn.Write([]byte(formatErrMsg(500, err.Error())))
		return
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

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

func formatErrMsg(status int, msg string) string {
	// 获取标准状态码的原因短语，如果没有找到，默认返回 "Unknown Status"
	reasonPhrase := http.StatusText(status)
	if reasonPhrase == "" {
		reasonPhrase = "Unknown Status"
	}

	// 计算消息体的长度
	contentLength := len(msg)

	// 格式化HTTP响应
	var errResponse = fmt.Sprintf("HTTP/1.1 %d %s\r\n"+
		"Content-Type: text/plain\r\n"+
		"Content-Length: %d\r\n"+
		"\r\n"+
		"%s",
		status, reasonPhrase, contentLength, msg)

	return errResponse
}
