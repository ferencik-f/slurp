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
	s := &server{token: "secret123"}
	r, _ := http.NewRequest("PUT", "/upload", nil)
	r.Header.Set("Authorization", "Bearer secret123")
	if !s.checkAuth(r) {
		t.Fatal("expected auth to pass with correct Bearer token")
	}
}

func TestCheckAuth_QueryParam(t *testing.T) {
	s := &server{token: "secret123"}
	r, _ := http.NewRequest("PUT", "/upload?token=secret123", nil)
	if !s.checkAuth(r) {
		t.Fatal("expected auth to pass with correct query token")
	}
}

func TestCheckAuth_WrongToken(t *testing.T) {
	s := &server{token: "secret123"}
	r, _ := http.NewRequest("PUT", "/upload?token=wrong", nil)
	if s.checkAuth(r) {
		t.Fatal("expected auth to fail with wrong token")
	}
}

func TestCheckAuth_Missing(t *testing.T) {
	s := &server{token: "secret123"}
	r, _ := http.NewRequest("PUT", "/upload", nil)
	if s.checkAuth(r) {
		t.Fatal("expected auth to fail with no token")
	}
}

func TestCheckAuth_HeaderTakesPrecedence(t *testing.T) {
	s := &server{token: "secret123"}
	r, _ := http.NewRequest("PUT", "/upload?token=wrong", nil)
	r.Header.Set("Authorization", "Bearer secret123")
	if !s.checkAuth(r) {
		t.Fatal("header should take precedence over query param")
	}
}

func TestHealthHandler(t *testing.T) {
	s := &server{}
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	s.healthHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	s := &server{}
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		req := httptest.NewRequest(method, "/health", nil)
		w := httptest.NewRecorder()
		s.healthHandler(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s /health: expected 405, got %d", method, w.Code)
		}
	}
}

func TestUploadHandler_Unauthorized(t *testing.T) {
	s := &server{token: "secret", dir: t.TempDir(), maxUpload: maxUploadBytes}
	req := httptest.NewRequest("PUT", "/upload", strings.NewReader("data"))
	w := httptest.NewRecorder()
	s.uploadHandler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUploadHandler_SavesFile(t *testing.T) {
	s := &server{token: "secret", dir: t.TempDir(), maxUpload: maxUploadBytes}
	req := httptest.NewRequest("PUT", "/upload?token=secret&filename=test.txt", strings.NewReader("hello world"))
	w := httptest.NewRecorder()
	s.uploadHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	content, err := os.ReadFile(filepath.Join(s.dir, "test.txt"))
	if err != nil {
		t.Fatalf("file not saved: %v", err)
	}
	if string(content) != "hello world" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestUploadHandler_BodyTooLarge(t *testing.T) {
	s := &server{token: "secret", dir: t.TempDir(), maxUpload: 5}
	body := strings.NewReader("this is more than 5 bytes")
	req := httptest.NewRequest("PUT", "/upload?token=secret&filename=big.bin", body)
	w := httptest.NewRecorder()
	s.uploadHandler(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}
}

func TestUploadHandler_BodyTooLargeLeavesRecoverableArtifact(t *testing.T) {
	s := &server{token: "secret", dir: t.TempDir(), maxUpload: 5}
	req := httptest.NewRequest("PUT", "/upload?token=secret&filename=big.bin", strings.NewReader("this is more than 5 bytes"))
	w := httptest.NewRecorder()

	s.uploadHandler(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}

	finalPath := filepath.Join(s.dir, "big.bin")
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatalf("expected no published file at %s, got err=%v", finalPath, err)
	}

	partPath := filepath.Join(s.dir, "big.bin.part")
	data, err := os.ReadFile(partPath)
	if err != nil {
		t.Fatalf("expected recoverable artifact at %s: %v", partPath, err)
	}
	if len(data) == 0 {
		t.Fatal("expected recoverable artifact to contain partial data")
	}
}

func TestUploadHandler_CollisionDeconflicts(t *testing.T) {
	s := &server{token: "secret", dir: t.TempDir(), maxUpload: maxUploadBytes}
	os.WriteFile(filepath.Join(s.dir, "file.txt"), []byte("original"), 0644)

	req := httptest.NewRequest("PUT", "/upload?token=secret&filename=file.txt", strings.NewReader("new"))
	w := httptest.NewRecorder()
	s.uploadHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if _, err := os.Stat(filepath.Join(s.dir, "file (1).txt")); err != nil {
		t.Fatal("expected deconflicted file 'file (1).txt' to exist")
	}
}
