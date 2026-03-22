package main

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseTunnelURL_Found(t *testing.T) {
	ch := make(chan string, 1)
	input := "INF  |  Your quick Tunnel has been created! Visit it at: https://abc-def-123.trycloudflare.com"
	go parseTunnelURL(strings.NewReader(input), ch)
	url := <-ch
	if url != "https://abc-def-123.trycloudflare.com" {
		t.Fatalf("unexpected URL: %q", url)
	}
}

func TestParseTunnelURL_NotFound(t *testing.T) {
	ch := make(chan string, 1)
	input := "some line without a URL\nanother line\n"
	parseTunnelURL(strings.NewReader(input), ch)
	if len(ch) != 0 {
		t.Fatal("expected no URL to be sent")
	}
}

func TestResolveBaseURL_NoTunnel(t *testing.T) {
	baseURL, cmd := resolveBaseURL(true, 8765, time.Millisecond, func(port int, ch chan<- string) (*exec.Cmd, error) {
		t.Fatal("launcher should not be called in no-tunnel mode")
		return nil, nil
	})

	if baseURL != "http://localhost:8765" {
		t.Fatalf("expected localhost fallback, got %q", baseURL)
	}
	if cmd != nil {
		t.Fatal("expected no tunnel command in no-tunnel mode")
	}
}

func TestResolveBaseURL_LaunchErrorFallsBackToLocalhost(t *testing.T) {
	baseURL, cmd := resolveBaseURL(false, 8765, time.Millisecond, func(port int, ch chan<- string) (*exec.Cmd, error) {
		return nil, errors.New("cloudflared missing")
	})

	if baseURL != "http://localhost:8765" {
		t.Fatalf("expected localhost fallback, got %q", baseURL)
	}
	if cmd != nil {
		t.Fatal("expected no tunnel command when launch fails")
	}
}

func TestResolveBaseURL_ExitedProcessFallsBackToLocalhost(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh to simulate cloudflared")
	}

	baseURL, cmd := resolveBaseURL(false, 8765, 50*time.Millisecond, func(port int, ch chan<- string) (*exec.Cmd, error) {
		cmd := exec.Command("sh", "-c", "exit 0")
		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start helper process: %v", err)
		}
		return cmd, nil
	})

	if baseURL != "http://localhost:8765" {
		t.Fatalf("expected localhost fallback, got %q", baseURL)
	}
	if cmd != nil {
		t.Fatal("expected no running tunnel command when process exits early")
	}
}

func TestResolveBaseURL_SilentTunnelFallsBackToLocalhost(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh to simulate cloudflared")
	}

	baseURL, cmd := resolveBaseURL(false, 8765, 20*time.Millisecond, func(port int, ch chan<- string) (*exec.Cmd, error) {
		cmd := exec.Command("sh", "-c", "sleep 5")
		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start helper process: %v", err)
		}
		return cmd, nil
	})

	if baseURL != "http://localhost:8765" {
		t.Fatalf("expected localhost fallback, got %q", baseURL)
	}
	if cmd != nil {
		t.Fatal("expected no running tunnel command after timeout fallback")
	}
}
