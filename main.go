package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
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
			fmt.Sscanf(p, "%d", &port)
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
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		uploadHandler(w, r, token, dir)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "server error:", err)
			os.Exit(1)
		}
	}()

	// Start tunnel or fall back to local URL
	urlCh := make(chan string, 1)
	if *noTunnel {
		urlCh <- fmt.Sprintf("http://localhost:%d", port)
	} else {
		if err := launchTunnel(port, urlCh); err != nil {
			fmt.Fprintf(os.Stderr, "cloudflared not found — local only mode\n")
			urlCh <- fmt.Sprintf("http://localhost:%d", port)
		}
	}

	// Print startup banner once URL is known
	go func() {
		baseURL := <-urlCh
		fmt.Println()
		fmt.Println("slurp is ready")
		fmt.Printf("Saving to: %s\n", dir)
		fmt.Println()
		fmt.Println("Push a file:")
		fmt.Printf("  curl -T <file> \"%s/upload?token=%s&filename=<file>\"\n", baseURL, token)
		fmt.Println()
		fmt.Println("Ctrl+C to stop.")
	}()

	// Graceful shutdown: first Ctrl+C warns if upload in progress, second force-quits
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	if atomic.LoadInt64(&activeUploads) > 0 {
		fmt.Fprintln(os.Stderr, "\nUpload in progress — press Ctrl+C again to force quit")
		<-sigCh
	}
	server.Close()
}
