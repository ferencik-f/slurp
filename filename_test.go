package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

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
