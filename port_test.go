package main

import (
	"fmt"
	"net"
	"testing"
)

func TestFindFreePort(t *testing.T) {
	port, err := findFreePort(8765)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port < 8765 {
		t.Fatalf("expected port >= 8765, got %d", port)
	}
	// Verify the returned port is actually bindable
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("port %d should be free: %v", port, err)
	}
	ln.Close()
}
