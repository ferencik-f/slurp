package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type reservedUpload struct {
	finalPath string
	partPath  string
	partFile  *os.File
}

func requestedFilename(r *http.Request) string {
	// 1. Path segment: /upload/filename.txt
	if after, ok := strings.CutPrefix(r.URL.Path, "/upload/"); ok && after != "" {
		return filepath.Base(after)
	}
	// 2. Query param: ?filename=foo.txt
	name := r.URL.Query().Get("filename")
	// 3. Timestamp fallback
	if name == "" {
		name = fmt.Sprintf("upload-%s.bin", time.Now().Format("20060102-150405"))
	}
	// Strip path components — prevents traversal attacks like ../../etc/passwd
	return filepath.Base(name)
}

func reserveUploadTarget(dir, requested string) (*reservedUpload, error) {
	for i := 0; ; i++ {
		finalPath := filepath.Join(dir, deconflictedName(requested, i))
		placeholder, err := os.OpenFile(finalPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return nil, err
		}
		if err := placeholder.Close(); err != nil {
			_ = os.Remove(finalPath)
			return nil, err
		}

		partPath := finalPath + ".part"
		partFile, err := os.OpenFile(partPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			_ = os.Remove(finalPath)
			if os.IsExist(err) {
				continue
			}
			return nil, err
		}

		return &reservedUpload{
			finalPath: finalPath,
			partPath:  partPath,
			partFile:  partFile,
		}, nil
	}
}

func deconflictedName(name string, suffix int) string {
	if suffix == 0 {
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	if base == "" {
		// Dotfile like .bashrc — treat whole name as base with no extension
		base = name
		ext = ""
	}
	return fmt.Sprintf("%s (%d)%s", base, suffix, ext)
}

func (target *reservedUpload) publish() error {
	if target.partFile != nil {
		if err := target.partFile.Close(); err != nil {
			target.partFile = nil
			_ = os.Remove(target.finalPath)
			return err
		}
		target.partFile = nil
	}

	if err := os.Rename(target.partPath, target.finalPath); err == nil {
		return nil
	} else if copyErr := copyFile(target.partPath, target.finalPath); copyErr != nil {
		_ = os.Remove(target.finalPath)
		return fmt.Errorf("publish upload: rename failed: %v; copy fallback failed: %w", err, copyErr)
	}

	_ = os.Remove(target.partPath)
	return nil
}

func (target *reservedUpload) keepPartial() {
	if target == nil {
		return
	}
	if target.partFile != nil {
		_ = target.partFile.Close()
		target.partFile = nil
	}
	_ = os.Remove(target.finalPath)
}

func (target *reservedUpload) cleanup() {
	if target == nil {
		return
	}
	if target.partFile != nil {
		_ = target.partFile.Close()
		target.partFile = nil
	}
	_ = os.Remove(target.partPath)
	_ = os.Remove(target.finalPath)
}

func copyFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}
	return dst.Close()
}
