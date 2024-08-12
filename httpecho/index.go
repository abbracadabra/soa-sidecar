package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func echoHandler(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "text/plain")

	// 打印请求方法、URL 和头部信息
	fmt.Fprintf(w, "Method: %s\n", r.Method)
	fmt.Fprintf(w, "URL: %s\n", r.URL.String())
	fmt.Fprintf(w, "Headers:\n")
	for key, values := range r.Header {
		for _, value := range values {
			fmt.Fprintf(w, "%s: %s\n", key, value)
		}
	}
	fmt.Fprintln(w, "\nBody:")

	// 将请求体原样返回
	io.Copy(w, r.Body)
}

func main() {
	// 将 /echo 路由关联到 echoHandler 函数
	http.HandleFunc("/", echoHandler)

	// 启动服务器并监听 8080 端口
	fmt.Println("Starting server on :8111")
	log.Fatal(http.ListenAndServe(":8111", nil))
}
