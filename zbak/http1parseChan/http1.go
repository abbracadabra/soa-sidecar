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
	"test/connManage"
	"test/utils/ioUtil"
	"time"
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
		err = handleServerInbound(req, conn)
		if err != nil {
			break
		}
	}
}

func handleServerInbound(r *http.Request, conn net.Conn) (retErr error) {

	defer r.Body.Close()
	// optional transcoding
	// clusterName := "example"
	// getCluster(clusterName) && start reading when create new conn

	cls := cluster.GetOrCreate("example")
	ins := cls.Choose()

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

	cliConn := ins.Pool.Get()

	disposeCliConn := false
	// 返回连接
	defer func() {
		ins.Pool.GiveBack(cliConn, disposeCliConn)
	}()

	//写请求
	se, de := sendCliReq(cliConn, req)
	if se != nil {
		disposeCliConn = true
		return se //直接关闭server连接
	}
	if de != nil {
		disposeCliConn = true
		io.Copy(io.Discard, req.Body)
		conn.Write([]byte(formatErrMsg(500, de.Error())))
		return nil
	}

	//等待响应
	cliRespFuture := connManage.AddReqFuture(cliConn, "1")
	select {
	case <-cliRespFuture.Finished:
	case <-time.After(time.Second * 10):
		disposeCliConn = true
		conn.Write([]byte(formatErrMsg(500, "timeout")))
		return nil
	}

	if cliRespFuture.Err != nil {
		disposeCliConn = true
		conn.Write([]byte(formatErrMsg(500, cliRespFuture.Err.Error())))
		return nil
	}
	//回写响应
	cliResp := cliRespFuture.Resp
	se, de = writebackServerConn(conn, cliResp.(*http.Response))
	if se != nil {
		_, de = io.Copy(io.Discard, cliResp.(*http.Response).Body)
		if de != nil {
			disposeCliConn = true
		}
		return se
	}
	if de != nil {
		disposeCliConn = true
		return de
	}
	return nil
}
func sendCliReq(cliConn net.Conn, req *http.Request) (error, error) {
	var se error
	_, de := fmt.Fprintf(cliConn, "%s %s HTTP/1.1\r\n", req.Method, req.URL.RequestURI())
	for key, values := range req.Header {
		for _, value := range values {
			_, de = fmt.Fprintf(cliConn, "%s: %s\r\n", textproto.CanonicalMIMEHeaderKey(key), value)
		}
	}
	_, de = fmt.Fprintf(cliConn, "\r\n")
	if req.Body != nil {
		defer req.Body.Close()
		_, se, de = ioUtil.Copy(cliConn, req.Body)
	}
	return se, de
}

func writebackServerConn(serverConn net.Conn, resp *http.Response) (error, error) {

	var se, de error
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	// optional transcoding
	_, se = serverConn.Write([]byte(resp.Proto + " " + resp.Status + "\r\n"))
	se = resp.Header.Write(serverConn)
	_, se = serverConn.Write([]byte("\r\n"))
	_, se, de = ioUtil.Copy(serverConn, resp.Body)
	return se, de
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
