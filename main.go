package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	portFlag := flag.Int("port", 0, "listen port (default: first free from 8765)")
	dirFlag := flag.String("dir", "", "upload directory (default: ~/Downloads/slurp)")
	tokenFlag := flag.String("token", "", "auth token (default: auto-generated)")
	noTunnel := flag.Bool("no-tunnel", false, "disable cloudflared tunnel")
	flag.Parse()

	// Resolve port: flag → env → auto
	port := *portFlag
	if port == 0 {
		if p := os.Getenv("PORT"); p != "" {
			v, err := strconv.Atoi(p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: invalid PORT %q: %v\n", p, err)
				os.Exit(1)
			}
			port = v
		}
	}
	if port == 0 {
		var err error
		port, err = findFreePort(8765)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}

	// Resolve upload dir: flag → env → ~/Downloads/slurp
	dir := *dirFlag
	if dir == "" {
		dir = os.Getenv("UPLOAD_DIR")
	}
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "Downloads", "slurp")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating upload dir:", err)
		os.Exit(1)
	}

	// Resolve token: flag → env → auto-generated
	token := *tokenFlag
	if token == "" {
		token = os.Getenv("UPLOAD_TOKEN")
	}
	if token == "" {
		token = generateToken()
	}

	// Register HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	uploadRoute := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		uploadHandler(w, r, token, dir)
	}
	mux.HandleFunc("/upload", uploadRoute)
	mux.HandleFunc("/upload/", uploadRoute)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "server error:", err)
			os.Exit(1)
		}
	}()

	// Start tunnel or fall back to local URL
	urlCh := make(chan string, 1)
	var tunnelCmd *exec.Cmd
	if *noTunnel {
		urlCh <- fmt.Sprintf("http://localhost:%d", port)
	} else {
		cmd, err := launchTunnel(port, urlCh)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cloudflared not found — local only mode\n")
			urlCh <- fmt.Sprintf("http://localhost:%d", port)
		} else {
			tunnelCmd = cmd
		}
	}

	// shutdownCh is closed when shutdown begins, allowing the banner goroutine to exit.
	shutdownCh := make(chan struct{})

	// Print startup banner once URL is known
	go func() {
		const (
			reset  = "\033[0m"
			bold   = "\033[1m"
			dim    = "\033[2m"
			cyan   = "\033[36m"
			green  = "\033[32m"
			yellow = "\033[33m"
		)
		sep := "  " + cyan + strings.Repeat("─", 62) + reset
		select {
		case baseURL := <-urlCh:
			curlCmd := fmt.Sprintf(`curl -T <file> "%s/upload/<file>?token=%s"`, baseURL, token)
			fmt.Println()
			fmt.Printf("  %sslurp%s  ·  ready\n", bold+green, reset)
			fmt.Println(sep)
			fmt.Println()
			fmt.Printf("  %sdir%s    %s\n", dim, reset, dir)
			fmt.Printf("  %stoken%s  %s%s%s\n", dim, reset, bold, token, reset)
			fmt.Println()
			fmt.Println(sep)
			fmt.Println()
			fmt.Printf("  %s%s%s\n", bold+yellow, curlCmd, reset)
			fmt.Println()
			fmt.Println(sep)
			fmt.Printf("  %s^C to quit%s\n", dim, reset)
			fmt.Println()
		case <-shutdownCh:
		}
	}()

	// Graceful shutdown: first Ctrl+C warns if upload in progress, second force-quits
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	if atomic.LoadInt64(&activeUploads) > 0 {
		fmt.Fprintln(os.Stderr, "\nUpload in progress — press Ctrl+C again to force quit")
		<-sigCh
	}

	close(shutdownCh)

	if tunnelCmd != nil {
		tunnelCmd.Process.Kill()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}
