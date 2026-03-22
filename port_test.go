package main

import (
	"testing"
)

func TestListenOnPort_BindsRequestedPort(t *testing.T) {
	ln, port, err := listenOnPort(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() {
		_ = ln.Close()
	})

	if port < defaultPortStart {
		t.Fatalf("expected port >= %d, got %d", defaultPortStart, port)
	}

	addr := listenerPort(ln)
	if addr != port {
		t.Fatalf("expected reported port %d to match listener port %d", port, addr)
	}
}

func TestListenOnPort_UsesExplicitPort(t *testing.T) {
	probe, explicitPort, err := listenOnPort(0)
	if err != nil {
		t.Fatalf("failed to allocate probe listener: %v", err)
	}
	_ = probe.Close()

	ln, port, err := listenOnPort(explicitPort)
	if err != nil {
		t.Fatalf("unexpected error binding explicit port: %v", err)
	}
	t.Cleanup(func() {
		_ = ln.Close()
	})

	if port != explicitPort {
		t.Fatalf("expected explicit port %d, got %d", explicitPort, port)
	}
}
