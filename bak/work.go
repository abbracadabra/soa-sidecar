package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

func main() {
	ln, err := net.Listen("tcp", ":8083")
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

		go handleConnection4(conn)
	}
}

func handleConnection4(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			fmt.Println("Error reading request:", err)
			return
		}

		// Output the request method, URL, and headers
		fmt.Println("Method:", req.Method)
		fmt.Println("URL:", req.URL)
		fmt.Println("Headers:", req.Header)

		fmt.Println("req.URL.Path:", req.URL.Path)
		fmt.Println("req.URL.Host:", req.URL.Host)
		fmt.Println("req.URL.Scheme:", req.URL.Scheme)
		fmt.Println("req.URL.RawQuery:", req.URL.RawQuery)
		fmt.Println("req.URL.RawQuery:", req.URL.Fragment)
		fmt.Println("req.URL.RawQuery:", req.URL.RawPath)
		fmt.Println("req.URL.RawQuery:", req.URL.Opaque)

		// Get the body as io.Reader
		body := req.Body

		// Process the body (example: just copy it to stdout)
		data, _ := io.ReadAll(body)
		str := string(data)

		// 替换不可见字符
		str = strings.Replace(str, "\r", "\\r", -1)
		str = strings.Replace(str, "\n", "\\n", -1)
		str = strings.Replace(str, "\t", "\\t", -1)
		fmt.Println("bbody " + str)

		// Send a basic response
		response := "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"
		conn.Write([]byte(response))
	}

}
