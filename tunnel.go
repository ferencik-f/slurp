package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"time"
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

func resolveBaseURL(noTunnel bool, port int, wait time.Duration, launch func(int, chan<- string) (*exec.Cmd, error)) (string, *exec.Cmd) {
	localURL := fmt.Sprintf("http://localhost:%d", port)
	if noTunnel {
		return localURL, nil
	}

	urlCh := make(chan string, 1)
	cmd, err := launch(port, urlCh)
	if err != nil {
		return localURL, nil
	}

	exitCh := make(chan error, 1)
	go func() {
		exitCh <- cmd.Wait()
	}()

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case url := <-urlCh:
		if url == "" {
			stopTunnelProcess(cmd, exitCh)
			return localURL, nil
		}
		return url, cmd
	case <-exitCh:
		return localURL, nil
	case <-timer.C:
		stopTunnelProcess(cmd, exitCh)
		return localURL, nil
	}
}

func stopTunnelProcess(cmd *exec.Cmd, exitCh <-chan error) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	_ = cmd.Process.Kill()

	select {
	case <-exitCh:
	case <-time.After(250 * time.Millisecond):
	}
}
