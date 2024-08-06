package main

import (
	"net"
)

// func main() {
// 	listener, err := net.Listen("tcp", ":your_port")
// 	if err != nil {
// 		log.Fatalf("Failed to listen: %v", err)
// 	}
// 	defer listener.Close()

// 	for {
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			log.Printf("Failed to accept connection: %v", err)
// 			continue
// 		}

// 		// 处理客户端连接
// 		go handleConnection(conn)
// 	}
// }

func handleConnection(conn net.Conn) {
	defer conn.Close()
	// 在这里处理客户端的请求
}
