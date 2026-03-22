package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
)

var tunnelURLRe = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// parseTunnelURL scans r line by line and sends the first trycloudflare URL to ch.
func parseTunnelURL(r io.Reader, ch chan<- string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if m := tunnelURLRe.FindString(scanner.Text()); m != "" {
			ch <- m
			return
		}
	}
}

// launchTunnel starts cloudflared and delivers the public URL via ch.
// Returns the running command (for later cleanup) or an error if cloudflared is not found in PATH.
func launchTunnel(port int, ch chan<- string) (*exec.Cmd, error) {
	bin, err := exec.LookPath("cloudflared")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(bin, "tunnel", "--url", fmt.Sprintf("http://localhost:%d", port))
	// cloudflared prints the URL to stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go parseTunnelURL(stderr, ch)
	return cmd, nil
}
