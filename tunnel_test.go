package main

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseTunnelURL_Found(t *testing.T) {
	ch := make(chan tunnelResult, 1)
	input := "INF  |  Your quick Tunnel has been created! Visit it at: https://abc-def-123.trycloudflare.com"
	go parseTunnelURL(strings.NewReader(input), ch)
	result := <-ch
	if result.err != nil {
		t.Fatalf("unexpected parse error: %v", result.err)
	}
	if result.url != "https://abc-def-123.trycloudflare.com" {
		t.Fatalf("unexpected URL: %q", result.url)
	}
}

func TestParseTunnelURL_NotFound(t *testing.T) {
	ch := make(chan tunnelResult, 1)
	input := "some line without a URL\nanother line\n"
	parseTunnelURL(strings.NewReader(input), ch)
	if len(ch) != 0 {
		t.Fatal("expected no URL to be sent")
	}
}

func TestParseTunnelURL_ReaderError(t *testing.T) {
	ch := make(chan tunnelResult, 1)
	parseTunnelURL(errReader{err: errors.New("boom")}, ch)

	result := <-ch
	if result.err == nil {
		t.Fatal("expected reader error to be reported")
	}
}

func TestResolveBaseURL_NoTunnel(t *testing.T) {
	baseURL, cmd, warn := resolveBaseURL(context.Background(), true, 8765, time.Millisecond, func(ctx context.Context, port int, ch chan<- tunnelResult) (*exec.Cmd, error) {
		t.Fatal("launcher should not be called in no-tunnel mode")
		return nil, nil
	})

	if baseURL != "http://localhost:8765" {
		t.Fatalf("expected localhost fallback, got %q", baseURL)
	}
	if cmd != nil {
		t.Fatal("expected no tunnel command in no-tunnel mode")
	}
	if warn != nil {
		t.Fatalf("expected no warning in no-tunnel mode, got %v", warn)
	}
}

func TestResolveBaseURL_LaunchErrorFallsBackToLocalhost(t *testing.T) {
	baseURL, cmd, warn := resolveBaseURL(context.Background(), false, 8765, time.Millisecond, func(ctx context.Context, port int, ch chan<- tunnelResult) (*exec.Cmd, error) {
		return nil, errors.New("cloudflared missing")
	})

	if baseURL != "http://localhost:8765" {
		t.Fatalf("expected localhost fallback, got %q", baseURL)
	}
	if cmd != nil {
		t.Fatal("expected no tunnel command when launch fails")
	}
	if warn == nil {
		t.Fatal("expected warning when launch fails")
	}
}

func TestResolveBaseURL_ExitedProcessFallsBackToLocalhost(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh to simulate cloudflared")
	}

	baseURL, cmd, warn := resolveBaseURL(context.Background(), false, 8765, 50*time.Millisecond, func(ctx context.Context, port int, ch chan<- tunnelResult) (*exec.Cmd, error) {
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
	if warn == nil {
		t.Fatal("expected warning when process exits early")
	}
}

func TestResolveBaseURL_SilentTunnelFallsBackToLocalhost(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh to simulate cloudflared")
	}

	baseURL, cmd, warn := resolveBaseURL(context.Background(), false, 8765, 20*time.Millisecond, func(ctx context.Context, port int, ch chan<- tunnelResult) (*exec.Cmd, error) {
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
	if warn == nil {
		t.Fatal("expected warning when tunnel times out")
	}
}

type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}
