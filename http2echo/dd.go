package main

import (
	"io"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

// EchoHandler echoes back the request body.
func EchoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		io.WriteString(w, "Hello, HTTP/2 without SSL!")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	w.Header().Set("Content-Type", "text/plain")
	w.Write(body)
}

func main() {
	http2Server := &http2.Server{}

	handler := http.HandlerFunc(EchoHandler)
	httpServer := &http.Server{
		Addr:    ":8010",
		Handler: handler,
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", httpServer.Addr, err)
	}

	log.Printf("HTTP/2 server is running at %s", httpServer.Addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go func() {
			http2Server.ServeConn(conn, &http2.ServeConnOpts{
				BaseConfig: httpServer,
			})
		}()
	}
}
