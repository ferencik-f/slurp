package main

import (
	"fmt"
	"net"
)

// findFreePort returns the first free TCP port starting from start.
// It tries to bind each port — if binding succeeds, the port is free.
func findFreePort(start int) (int, error) {
	for port := start; port < start+100; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port found in range %d-%d", start, start+99)
}
