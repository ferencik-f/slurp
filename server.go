package main

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

// maxUploadBytes is the maximum accepted request body size (2 GiB by default).
var maxUploadBytes int64 = 2 << 30

var activeUploads int64

func checkAuth(r *http.Request, token string) bool {
	tb := []byte(token)
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(auth, "Bearer ")), tb) == 1
	}
	return subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("token")), tb) == 1
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

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	dest := resolveFilename(r, dir)
	f, err := os.Create(dest)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := io.Copy(f, r.Body); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "Failed to write file", http.StatusInternalServerError)
		}
		return
	}

	fmt.Fprintf(w, "Saved: %s\n", filepath.Base(dest))
}
