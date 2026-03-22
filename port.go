package main

import (
	"fmt"
	"net"
)

const (
	defaultPortStart      = 8765
	defaultPortSearchSpan = 100
)

func listenOnPort(port int) (net.Listener, int, error) {
	if port != 0 {
		return listenTCP(port)
	}

	for candidate := defaultPortStart; candidate < defaultPortStart+defaultPortSearchSpan; candidate++ {
		ln, actualPort, err := listenTCP(candidate)
		if err == nil {
			return ln, actualPort, nil
		}
	}

	return nil, 0, fmt.Errorf(
		"no free port found in range %d-%d",
		defaultPortStart,
		defaultPortStart+defaultPortSearchSpan-1,
	)
}

func listenTCP(port int) (net.Listener, int, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, 0, err
	}
	return ln, listenerPort(ln), nil
}

func listenerPort(ln net.Listener) int {
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return 0
	}
	return addr.Port
}
