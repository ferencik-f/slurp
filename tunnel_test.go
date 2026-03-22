package main

import (
	"strings"
	"testing"
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
