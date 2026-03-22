package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFilename_PathSegment(t *testing.T) {
	dir := t.TempDir()
	r, _ := http.NewRequest("PUT", "/upload/receiver.py", nil)
	got := resolveFilename(r, dir)
	if filepath.Base(got) != "receiver.py" {
		t.Fatalf("expected receiver.py, got %s", got)
	}
}

func TestResolveFilename_PathSegmentTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	r, _ := http.NewRequest("PUT", "/upload/path.txt", nil)
	r.URL.RawQuery = "filename=query.txt"
	got := resolveFilename(r, dir)
	if filepath.Base(got) != "path.txt" {
		t.Fatalf("path segment should win over query param, got %s", got)
	}
}

func TestResolveFilename_QueryParam(t *testing.T) {
	dir := t.TempDir()
	r, _ := http.NewRequest("PUT", "/upload?filename=photo.jpg", nil)
	got := resolveFilename(r, dir)
	if filepath.Base(got) != "photo.jpg" {
		t.Fatalf("expected photo.jpg, got %s", got)
	}
}

func TestResolveFilename_Fallback(t *testing.T) {
	dir := t.TempDir()
	r, _ := http.NewRequest("PUT", "/upload", nil)
	got := resolveFilename(r, dir)
	base := filepath.Base(got)
	if len(base) < 10 {
		t.Fatalf("expected timestamp fallback filename, got %q", base)
	}
}

func TestDeconflict_Dotfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".bashrc")
	os.WriteFile(path, []byte("x"), 0644)

	got := deconflict(path)
	want := filepath.Join(dir, ".bashrc (1)")
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestDeconflict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")

	// No collision — return path as-is
	got := deconflict(path)
	if got != path {
		t.Fatalf("expected %s, got %s", path, got)
	}

	os.WriteFile(path, []byte("x"), 0644)

	// Collision with original → (1) suffix
	got = deconflict(path)
	want := filepath.Join(dir, "file (1).txt")
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}

	os.WriteFile(want, []byte("x"), 0644)

	// Collision with (1) → (2) suffix
	got = deconflict(path)
	want2 := filepath.Join(dir, "file (2).txt")
	if got != want2 {
		t.Fatalf("expected %s, got %s", want2, got)
	}
}
