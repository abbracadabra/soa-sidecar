package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

type Header struct {
	Key   string
	Value string
}
type HeadersList []Header

func (h HeadersList) getFirstKey(key string) (string, bool) {
	for _, header := range h {
		if header.Key == key {
			return header.Value, true
		}
	}
	return "", false
}

var methodsNoBody = map[string]bool{
	"CONNECT": true,
	"GET":     true,
	"HEAD":    true,
}

func readChunkedBody(reader *bufio.Reader) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		for {
			// 读取每块的大小
			line, err := reader.ReadString('\n')
			if err != nil {
				pw.CloseWithError(err)
				return
			}

			// 将块大小行写入缓冲区
			pw.Write([]byte(line))

			// 解析块的大小
			var size int
			_, err = fmt.Sscanf(line, "%x", &size)
			if err != nil {
				pw.CloseWithError(err)
				return
			}

			// 如果块大小为0，则表示结束
			if size == 0 {
				break
			}

			// 读取指定大小的数据块
			chunk := make([]byte, size)
			_, err = io.ReadFull(reader, chunk)
			if err != nil {
				pw.CloseWithError(err)
				return
			}

			// 将数据块写入缓冲区
			pw.Write(chunk)

			// 读取并保留CRLF
			crlf := make([]byte, 2)
			_, err = io.ReadFull(reader, crlf)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			pw.Write(crlf)
		}

		// 读取并保留最后的CRLF
		lastCRLF := make([]byte, 2)
		_, err := io.ReadFull(reader, lastCRLF)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Write(lastCRLF)
	}()

	return pr
}

func parseHTTPHeaders(reader *bufio.Reader) *HeadersList {
	var serverHeaders HeadersList

	for {
		headerLine, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading header line:", err)
			return nil
		}
		headerLine = strings.TrimSpace(headerLine) // 移除首尾空白字符

		if headerLine == "" {
			break // 空行表示请求头结束
		}

		// 将请求头分割为键和值
		parts := strings.SplitN(headerLine, ":", 2)
		if len(parts) != 2 {
			fmt.Println("Invalid header line:", headerLine)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 将键值对添加到列表中
		serverHeaders = append(serverHeaders, Header{Key: key, Value: value})

	}
	return &serverHeaders
}

func handleConnection(serverConn net.Conn) {
	defer serverConn.Close()

	serverConnReader := bufio.NewReader(serverConn)

	for {
		serverRequest := parseRequest(serverConnReader)

		//todo proxy
		//create interaction
		//Todo match route || choose host
		cliConn, err := net.Dial("tcp", "127.0.0.1:80")
		defer cliConn.Close()

		cliRequest := fmt.Sprintf("%s %s HTTP/1.1\r\n", serverRequest.method, serverRequest.path)
		for _, header := range *serverRequest.headers {
			cliRequest += fmt.Sprintf("%s: %s\r\n", header.Key, header.Value)
		}
		cliRequest += "\r\n"

		// 发送请求头和空行
		_, err = cliConn.Write([]byte(cliRequest))
		if err != nil {
			fmt.Println("Error sending request headers:", err)
			return
		}
		// 发送请求体
		if serverRequest.bodyReader != nil {
			_, err = io.Copy(cliConn, serverRequest.bodyReader)
			if err != nil {
				fmt.Println("Error sending request body:", err)
				return
			}
		}
		// 读取响应
		cliConnReader := bufio.NewReader(cliConn)

		cliResp := parseResponse(cliConnReader)

		statusLineStr := fmt.Sprintf("%s %d %s\r\n", cliResp.version, cliResp.status, cliResp.reason)
		var headerBuilder strings.Builder
		for _, header := range *cliResp.headers {
			headerBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", header.Key, header.Value))
		}

		//头
		_, err = serverConn.Write([]byte(statusLineStr + headerBuilder.String() + "\r\n"))
		if err != nil {
			fmt.Println("发送响应头错误:", err)
			return
		}
		//body
		_, err = io.Copy(serverConn, cliResp.bodyReader)
		if err != nil {
			fmt.Println("发送响应体错误:", err)
			return
		}
	}
}

func parseRequest(reader *bufio.Reader) *Request {

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading:", err.Error())
		return nil
	}

	// 使用空格分割请求行
	parts := strings.SplitN(requestLine, " ", 3)

	// 检查是否有足够的部分
	if len(parts) < 3 {
		fmt.Println("无效的HTTP请求行")
		return nil
	}

	// 获取HTTP方法
	method := parts[0]

	// 获取请求的资源路径
	path := parts[1]

	// 获取HTTP版本
	version := parts[2]

	headers := parseHTTPHeaders(reader)

	var bodyLen int64 = -1
	if bodyLenStr, found := headers.getFirstKey("Content-Length"); found {
		len, err := strconv.ParseInt(bodyLenStr, 10, 64)
		if err != nil || len < 0 {
			return nil
		}
		bodyLen = len
	}

	transferEncoding, _ := headers.getFirstKey("Transfer-Encoding")
	contentType, _ := headers.getFirstKey("Content-Type")

	var bodyReader io.Reader
	if bodyLen >= 0 {
		bodyReader = io.LimitReader(reader, bodyLen)
	} else if transferEncoding == "chunked" {
		chunkedReader := readChunkedBody(reader)
		bodyReader = chunkedReader
	} else if contentType == "" && methodsNoBody[method] {
		// no body
	} else {
		//throw
		return nil
	}

	return &Request{method: method, path: path, version: version, headers: headers, bodyReader: bodyReader}
}

func parseResponse(reader *bufio.Reader) *Response {

	statusLine, err := reader.ReadString('\n')

	if err != nil {
		fmt.Println("Error reading:", err.Error())
		return nil
	}

	parts := strings.SplitN(statusLine, " ", 3)
	if len(parts) != 3 {
		return nil
	}

	i, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil
	}

	headers := parseHTTPHeaders(reader)

	var bodyLen int64 = -1
	if bodyLenStr, found := headers.getFirstKey("Content-Length"); found {
		len, err := strconv.ParseInt(bodyLenStr, 10, 64)
		if err != nil || len < 0 {
			return nil
		}
		bodyLen = len
	}
	transferEncoding, _ := headers.getFirstKey("Transfer-Encoding")
	contentType, _ := headers.getFirstKey("Content-Type")

	var bodyReader io.Reader
	if bodyLen >= 0 {
		bodyReader = io.LimitReader(reader, bodyLen)
	} else if transferEncoding == "chunked" {
		chunkedReader := readChunkedBody(reader)
		bodyReader = chunkedReader
	} else if contentType == "" {
		// no body
	} else {
		//throw
		return nil
	}

	return &Response{version: parts[0], status: i, reason: parts[2], headers: headers, bodyReader: bodyReader}
}

type Request struct {
	method     string
	path       string
	version    string
	headers    *HeadersList
	bodyReader io.Reader
}

type Response struct {
	version    string
	status     int
	reason     string
	headers    *HeadersList
	bodyReader io.Reader
}

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server:", err.Error())
		return
	}
	defer ln.Close()

	fmt.Println("HTTP server listening on port 8080")

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			continue
		}

		go handleConnection(conn)
	}
}
