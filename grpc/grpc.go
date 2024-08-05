package main

import (
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
)

type ConnPool struct {
	mu        sync.Mutex
	available []*grpc.ClientConn
	inUse     map[*grpc.ClientConn]bool
}

func NewConnPool() *ConnPool {
	return &ConnPool{
		inUse: make(map[*grpc.ClientConn]bool),
	}
}

func (p *ConnPool) GetConn(target string) (*grpc.ClientConn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.available {
		if !p.inUse[conn] {
			p.inUse[conn] = true
			return conn, nil
		}
	}

	// No available connections, create a new one
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	p.available = append(p.available, conn)
	p.inUse[conn] = true
	return conn, nil
}

func (p *ConnPool) ReturnConn(conn *grpc.ClientConn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.inUse[conn] = false
}

var hostPools = map[string]*ConnPool{}
var globalMutex sync.Mutex

func main() {
	// Assuming you have a mechanism to accept TCP connections.
	listener, err := net.Listen("tcp", ":5000")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

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

	// Extract host from the connection's context or metadata.
	// For simplicity, we'll use a placeholder function to get the host.
	host := extractHost(conn)

	// Lock to safely access and modify the hostPools map
	globalMutex.Lock()
	pool, exists := hostPools[host]
	if !exists {
		pool = NewConnPool()
		hostPools[host] = pool
	}
	globalMutex.Unlock()

	targetAddress := determineTarget(host)
	if targetAddress == "" {
		log.Printf("No target found for host: %s", host)
		return
	}

	// Get a connection from the pool
	clientConn, err := pool.GetConn(targetAddress)
	if err != nil {
		log.Printf("Failed to get connection: %v", err)
		return
	}

	// Use the clientConn to forward the gRPC call
	err = forwardRequest(conn, clientConn)
	if err != nil {
		log.Printf("Failed to forward request: %v", err)
	}

	// Return the connection to the pool
	pool.ReturnConn(clientConn)
}

func extractHost(conn net.Conn) string {
	// Placeholder function to extract the host from the connection.
	// In a real scenario, this could involve parsing metadata or
	// reading from the connection context.
	return "exampleHost"
}

func determineTarget(host string) string {
	// Implement your host-to-target mapping logic
	if host == "exampleHost" {
		return "localhost:50051"
	}
	return ""
}

func forwardRequest(conn net.Conn, clientConn *grpc.ClientConn) error {
	// Implement the logic to forward the request from conn
	// to clientConn and send the response back.
	// This typically involves creating a gRPC client stub and
	// using it to make the request.

	// For demonstration purposes, let's assume we have a function
	// called `proxyCall` that handles the request forwarding.
	return proxyCall(conn, clientConn)
}

func proxyCall(conn net.Conn, clientConn *grpc.ClientConn) error {
	// Example function to demonstrate request forwarding.
	// Implement the actual gRPC call forwarding logic here.
	return nil
}
