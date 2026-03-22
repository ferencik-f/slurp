package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	readHeaderTimeout = 10 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
	tunnelWaitTimeout = 15 * time.Second
	shutdownTimeout   = 5 * time.Second
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	cfg, err := loadConfig(args, os.Getenv, os.UserHomeDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return fmt.Errorf("create upload dir: %w", err)
	}

	listener, port, err := listenOnPort(cfg.Port)
	if err != nil {
		return err
	}
	defer listener.Close()

	srv := newServer(cfg.Token, cfg.Dir)
	httpServer := &http.Server{
		Handler:           newMux(srv),
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	serveErrCh := make(chan error, 1)
	go func() {
		err := httpServer.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			serveErrCh <- err
			return
		}
		close(serveErrCh)
	}()

	tunnelCtx, cancelTunnel := context.WithCancel(context.Background())
	defer cancelTunnel()

	if !cfg.NoTunnel {
		fmt.Fprintln(stdout, "  starting tunnel...")
	}
	baseURL, tunnelCmd, tunnelWarn := resolveBaseURL(tunnelCtx, cfg.NoTunnel, port, tunnelWaitTimeout, launchTunnel)
	if tunnelWarn != nil {
		fmt.Fprintf(stderr, "warning: %v\n", tunnelWarn)
	}
	printReadyBanner(stdout, baseURL, cfg.Token, cfg.Dir)

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err := <-serveErrCh:
		cancelTunnel()
		if tunnelCmd != nil {
			stopTunnelProcess(tunnelCmd, nil)
		}
		if err != nil {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	case <-sigCh:
	}

	if srv.activeUploads.Load() > 0 {
		fmt.Fprintln(stderr, "\nUpload in progress - press Ctrl+C again to force quit")
		select {
		case err := <-serveErrCh:
			if err != nil {
				return fmt.Errorf("serve: %w", err)
			}
			return nil
		case <-sigCh:
		}
	}

	if tunnelCmd != nil {
		cancelTunnel()
		stopTunnelProcess(tunnelCmd, nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err, ok := <-serveErrCh; ok && err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

func isColorTerminal(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := file.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func printReadyBanner(w io.Writer, baseURL, token, dir string) {
	reset, bold, dim, cyan, green, yellow := "", "", "", "", "", ""
	if isColorTerminal(w) {
		reset = "\033[0m"
		bold = "\033[1m"
		dim = "\033[2m"
		cyan = "\033[36m"
		green = "\033[32m"
		yellow = "\033[33m"
	}

	sep := "  " + cyan + strings.Repeat("─", 62) + reset
	curlCmd := fmt.Sprintf(`curl -T <file> -H "Authorization: Bearer %s" "%s/upload/<file>"`, token, baseURL)

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %sslurp%s  ·  ready\n", bold+green, reset)
	fmt.Fprintln(w, sep)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %sdir%s    %s\n", dim, reset, dir)
	fmt.Fprintf(w, "  %stoken%s  %s%s%s\n", dim, reset, bold, token, reset)
	fmt.Fprintln(w)
	fmt.Fprintln(w, sep)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s%s%s\n", bold+yellow, curlCmd, reset)
	fmt.Fprintln(w)
	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "  %s^C to quit%s\n", dim, reset)
	fmt.Fprintln(w)
}
