package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var deconflictMu sync.Mutex

func resolveFilename(r *http.Request, dir string) string {
	// 1. Path segment: /upload/filename.txt
	if after, ok := strings.CutPrefix(r.URL.Path, "/upload/"); ok && after != "" {
		return deconflict(filepath.Join(dir, filepath.Base(after)))
	}
	// 2. Query param: ?filename=foo.txt
	name := r.URL.Query().Get("filename")
	// 3. Timestamp fallback
	if name == "" {
		name = fmt.Sprintf("upload-%s.bin", time.Now().Format("20060102-150405"))
	}
	// Strip path components — prevents traversal attacks like ../../etc/passwd
	name = filepath.Base(name)
	return deconflict(filepath.Join(dir, name))
}

func deconflict(path string) string {
	deconflictMu.Lock()
	defer deconflictMu.Unlock()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	dir := filepath.Dir(path)
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	if base == "" {
		// Dotfile like .bashrc — treat whole name as base with no extension
		base = name
		ext = ""
	}
	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
