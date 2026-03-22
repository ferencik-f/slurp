package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func resolveFilename(r *http.Request, dir string) string {
	name := r.URL.Query().Get("filename")
	if name == "" {
		name = fmt.Sprintf("upload-%s.bin", time.Now().Format("20060102-150405"))
	}
	// Strip path components — prevents traversal attacks like ../../etc/passwd
	name = filepath.Base(name)
	return deconflict(filepath.Join(dir, name))
}

func deconflict(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
