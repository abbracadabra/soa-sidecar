package main

import (
	"bytes"
	"fmt"
	"log"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

func main() {
	// Example data for an HTTP/2 headers frame
	data := []byte{
		0x00, 0x00, 0x12, // 3-byte payload length (18 bytes)
		0x01,                   // Type: HEADERS
		0x04,                   // Flags: END_HEADERS
		0x00, 0x00, 0x00, 0x01, // Stream identifier: 1
		// Payload (Header Block Fragment - for simplicity, not a real one)
		0x82, 0x86, 0x84, 0x41, 0x9c, 0x10, 0x0b, 0x5f, 0x08, 0x03,
		0x6c, 0x4f, 0x1e, 0x50, 0x06, 0x48, 0x0f, 0x40,
	}

	// Parse the frame
	framer := http2.NewFramer(nil, bytes.NewReader(data))
	frame, err := framer.ReadFrame()
	if err != nil {
		log.Fatalf("Error reading frame: %v", err)
	}

	// Assert the frame type
	headersFrame, ok := frame.(*http2.HeadersFrame)
	if !ok {
		log.Fatalf("Expected HeadersFrame, got %T", frame)
	}

	// Decode the header block fragment
	decoder := hpack.NewDecoder(4096, func(f hpack.HeaderField) {
		fmt.Printf("Header: %s: %s\n", f.Name, f.Value)
	})

	_, err = decoder.Write(headersFrame.HeaderBlockFragment())
	if err != nil {
		log.Fatalf("Error decoding headers: %v", err)
	}

	// Close the decoder to flush any remaining headers
	err = decoder.Close()
	if err != nil {
		log.Fatalf("Error closing decoder: %v", err)
	}
}
