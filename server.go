package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

var activeUploads int64

func checkAuth(r *http.Request, token string) bool {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ") == token
	}
	return r.URL.Query().Get("token") == token
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func uploadHandler(w http.ResponseWriter, r *http.Request, token, dir string) {
	if !checkAuth(r, token) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Track in-flight uploads for graceful shutdown
	atomic.AddInt64(&activeUploads, 1)
	defer atomic.AddInt64(&activeUploads, -1)

	dest := resolveFilename(r, dir)
	f, err := os.Create(dest)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := io.Copy(f, r.Body); err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Saved: %s\n", filepath.Base(dest))
}
