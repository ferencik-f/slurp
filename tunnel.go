package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"time"
)

var tunnelURLRe = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

type tunnelResult struct {
	url string
	err error
}

// parseTunnelURL scans r line by line and sends the first trycloudflare URL to ch.
func parseTunnelURL(r io.Reader, ch chan<- tunnelResult) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if m := tunnelURLRe.FindString(scanner.Text()); m != "" {
			ch <- tunnelResult{url: m}
			return
		}
	}
	if err := scanner.Err(); err != nil {
		ch <- tunnelResult{err: err}
	}
}

// launchTunnel starts cloudflared and delivers the public URL via ch.
// Returns the running command (for later cleanup) or an error if cloudflared is not found in PATH.
func launchTunnel(ctx context.Context, port int, ch chan<- tunnelResult) (*exec.Cmd, error) {
	bin, err := exec.LookPath("cloudflared")
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, bin, "tunnel", "--url", fmt.Sprintf("http://localhost:%d", port))
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

func resolveBaseURL(
	ctx context.Context,
	noTunnel bool,
	port int,
	wait time.Duration,
	launch func(context.Context, int, chan<- tunnelResult) (*exec.Cmd, error),
) (string, *exec.Cmd, error) {
	localURL := fmt.Sprintf("http://localhost:%d", port)
	if noTunnel {
		return localURL, nil, nil
	}

	urlCh := make(chan tunnelResult, 1)
	cmd, err := launch(ctx, port, urlCh)
	if err != nil {
		return localURL, nil, fmt.Errorf("cloudflared launch failed: %w", err)
	}

	exitCh := make(chan error, 1)
	go func() {
		exitCh <- cmd.Wait()
	}()

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case result := <-urlCh:
		if result.err != nil {
			stopTunnelProcess(cmd, exitCh)
			return localURL, nil, fmt.Errorf("cloudflared output failed: %w", result.err)
		}
		if result.url == "" {
			stopTunnelProcess(cmd, exitCh)
			return localURL, nil, errors.New("cloudflared did not report a public URL")
		}
		return result.url, cmd, nil
	case <-exitCh:
		return localURL, nil, errors.New("cloudflared exited before reporting a public URL")
	case <-timer.C:
		stopTunnelProcess(cmd, exitCh)
		return localURL, nil, fmt.Errorf("cloudflared did not report a public URL within %s", wait)
	case <-ctx.Done():
		stopTunnelProcess(cmd, exitCh)
		return localURL, nil, ctx.Err()
	}
}

func stopTunnelProcess(cmd *exec.Cmd, exitCh <-chan error) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	_ = cmd.Process.Kill()
	if exitCh == nil {
		return
	}

	select {
	case <-exitCh:
	case <-time.After(250 * time.Millisecond):
	}
}
