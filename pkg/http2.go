package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
)

const (
	// HTTP/2 帧类型
	frameData         = 0x0
	frameHeaders      = 0x1
	framePriority     = 0x2
	frameRstStream    = 0x3
	frameSettings     = 0x4
	framePushPromise  = 0x5
	framePing         = 0x6
	frameGoAway       = 0x7
	frameWindowUpdate = 0x8
	frameContinuation = 0x9
)

func main() {
	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer listen.Close()

	fmt.Println("Listening on :8080")
	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
			os.Exit(1)
		}
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	// 读取并解析 HTTP/2 帧
	for {
		// 读取帧头（9 个字节）
		header := make([]byte, 9)
		_, err := io.ReadFull(conn, header)
		if err != nil {
			fmt.Println("Error reading frame header:", err.Error())
			return
		}

		// 解析帧头
		length := binary.BigEndian.Uint32(append([]byte{0}, header[0:3]...))
		frameType := header[3]
		flags := header[4]
		streamID := binary.BigEndian.Uint32(header[5:9]) & 0x7FFFFFFF

		// 读取帧载荷
		payload := make([]byte, length)
		_, err = io.ReadFull(conn, payload)
		if err != nil {
			fmt.Println("Error reading frame payload:", err.Error())
			return
		}

		// 处理不同类型的帧
		switch frameType {
		case frameData:
			fmt.Println("Received DATA frame")
			// 处理 DATA 帧
		case frameHeaders:
			fmt.Println("Received HEADERS frame")
			// 处理 HEADERS 帧
		case framePriority:
			fmt.Println("Received PRIORITY frame")
			// 处理 PRIORITY 帧
		case frameRstStream:
			fmt.Println("Received RST_STREAM frame")
			// 处理 RST_STREAM 帧
		case frameSettings:
			fmt.Println("Received SETTINGS frame")
			// 处理 SETTINGS 帧
		case framePushPromise:
			fmt.Println("Received PUSH_PROMISE frame")
			// 处理 PUSH_PROMISE 帧
		case framePing:
			fmt.Println("Received PING frame")
			// 处理 PING 帧
		case frameGoAway:
			fmt.Println("Received GOAWAY frame")
			// 处理 GOAWAY 帧
		case frameWindowUpdate:
			fmt.Println("Received WINDOW_UPDATE frame")
			// 处理 WINDOW_UPDATE 帧
		case frameContinuation:
			fmt.Println("Received CONTINUATION frame")
			// 处理 CONTINUATION 帧
		default:
			fmt.Println("Received unknown frame type")
		}
	}
}
