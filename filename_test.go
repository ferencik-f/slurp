package main

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestRequestedFilename_PathSegment(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload/receiver.py", nil)
	if got := requestedFilename(r); got != "receiver.py" {
		t.Fatalf("expected receiver.py, got %s", got)
	}
}

func TestRequestedFilename_PathSegmentTakesPrecedence(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload/path.txt", nil)
	r.URL.RawQuery = "filename=query.txt"
	if got := requestedFilename(r); got != "path.txt" {
		t.Fatalf("path segment should win over query param, got %s", got)
	}
}

func TestRequestedFilename_QueryParam(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload?filename=photo.jpg", nil)
	if got := requestedFilename(r); got != "photo.jpg" {
		t.Fatalf("expected photo.jpg, got %s", got)
	}
}

func TestRequestedFilename_Fallback(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload", nil)
	got := requestedFilename(r)
	if len(got) < 10 {
		t.Fatalf("expected timestamp fallback filename, got %q", got)
	}
}

func TestReserveUploadTarget_Dotfile(t *testing.T) {
	dir := t.TempDir()
	target, err := reserveUploadTarget(dir, ".bashrc")
	if err != nil {
		t.Fatalf("unexpected error reserving first dotfile target: %v", err)
	}
	defer target.cleanup()

	next, err := reserveUploadTarget(dir, ".bashrc")
	if err != nil {
		t.Fatalf("unexpected error reserving second dotfile target: %v", err)
	}
	defer next.cleanup()

	want := filepath.Join(dir, ".bashrc (1)")
	if next.finalPath != want {
		t.Fatalf("expected %s, got %s", want, next.finalPath)
	}
}

func TestReserveUploadTarget_ConcurrentReservationsAreUnique(t *testing.T) {
	dir := t.TempDir()
	type result struct {
		target *reservedUpload
		err    error
	}

	results := make(chan result, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			target, err := reserveUploadTarget(dir, "file.txt")
			results <- result{target: target, err: err}
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var reserved []*reservedUpload
	for result := range results {
		if result.err != nil {
			t.Fatalf("unexpected reservation error: %v", result.err)
		}
		reserved = append(reserved, result.target)
	}
	defer func() {
		for _, target := range reserved {
			target.cleanup()
		}
	}()

	if len(reserved) != 2 {
		t.Fatalf("expected two reservations, got %d", len(reserved))
	}
	if reserved[0].finalPath == reserved[1].finalPath {
		t.Fatalf("expected unique final paths, got %s", reserved[0].finalPath)
	}
	for _, target := range reserved {
		if _, err := os.Stat(target.finalPath); err != nil {
			t.Fatalf("expected reserved final placeholder %s: %v", target.finalPath, err)
		}
		if _, err := os.Stat(target.partPath); err != nil {
			t.Fatalf("expected reserved partial artifact %s: %v", target.partPath, err)
		}
	}
}
