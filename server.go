package main

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
)

// maxUploadBytes is the default maximum accepted request body size (2 GiB).
const maxUploadBytes int64 = 2 << 30

type server struct {
	token         string
	dir           string
	maxUpload     int64
	activeUploads atomic.Int64
}

func newServer(token, dir string) *server {
	return &server{token: token, dir: dir, maxUpload: maxUploadBytes}
}

func (s *server) checkAuth(r *http.Request) bool {
	tb := []byte(s.token)
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(auth, "Bearer ")), tb) == 1
	}
	return subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("token")), tb) == 1
}

func (s *server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func (s *server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.checkAuth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Track in-flight uploads for graceful shutdown
	s.activeUploads.Add(1)
	defer s.activeUploads.Add(-1)

	r.Body = http.MaxBytesReader(w, r.Body, s.maxUpload)

	target, err := reserveUploadTarget(s.dir, requestedFilename(r))
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(target.partFile, r.Body); err != nil {
		target.keepPartial()
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "Failed to write file", http.StatusInternalServerError)
		}
		return
	}

	if err := target.publish(); err != nil {
		target.keepPartial()
		http.Error(w, "Failed to publish file", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Saved: %s\n", filepath.Base(target.finalPath))
}

func newMux(s *server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/upload", s.uploadHandler)
	mux.HandleFunc("/upload/", s.uploadHandler)
	return mux
}
