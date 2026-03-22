package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckAuth_BearerHeader(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload", nil)
	r.Header.Set("Authorization", "Bearer secret123")
	if !checkAuth(r, "secret123") {
		t.Fatal("expected auth to pass with correct Bearer token")
	}
}

func TestCheckAuth_QueryParam(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload?token=secret123", nil)
	if !checkAuth(r, "secret123") {
		t.Fatal("expected auth to pass with correct query token")
	}
}

func TestCheckAuth_WrongToken(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload?token=wrong", nil)
	if checkAuth(r, "secret123") {
		t.Fatal("expected auth to fail with wrong token")
	}
}

func TestCheckAuth_Missing(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload", nil)
	if checkAuth(r, "secret123") {
		t.Fatal("expected auth to fail with no token")
	}
}

func TestCheckAuth_HeaderTakesPrecedence(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload?token=wrong", nil)
	r.Header.Set("Authorization", "Bearer secret123")
	if !checkAuth(r, "secret123") {
		t.Fatal("header should take precedence over query param")
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	healthHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestUploadHandler_Unauthorized(t *testing.T) {
	dir := t.TempDir()
	req := httptest.NewRequest("PUT", "/upload", strings.NewReader("data"))
	w := httptest.NewRecorder()
	uploadHandler(w, req, "secret", dir)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUploadHandler_SavesFile(t *testing.T) {
	dir := t.TempDir()
	req := httptest.NewRequest("PUT", "/upload?token=secret&filename=test.txt", strings.NewReader("hello world"))
	w := httptest.NewRecorder()
	uploadHandler(w, req, "secret", dir)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	content, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatalf("file not saved: %v", err)
	}
	if string(content) != "hello world" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestUploadHandler_BodyTooLarge(t *testing.T) {
	orig := maxUploadBytes
	maxUploadBytes = 5
	defer func() { maxUploadBytes = orig }()

	dir := t.TempDir()
	body := strings.NewReader("this is more than 5 bytes")
	req := httptest.NewRequest("PUT", "/upload?token=secret&filename=big.bin", body)
	w := httptest.NewRecorder()
	uploadHandler(w, req, "secret", dir)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}
}

func TestUploadHandler_CollisionDeconflicts(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("original"), 0644)

	req := httptest.NewRequest("PUT", "/upload?token=secret&filename=file.txt", strings.NewReader("new"))
	w := httptest.NewRecorder()
	uploadHandler(w, req, "secret", dir)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if _, err := os.Stat(filepath.Join(dir, "file (1).txt")); err != nil {
		t.Fatal("expected deconflicted file 'file (1).txt' to exist")
	}
}