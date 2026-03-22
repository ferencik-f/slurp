package main

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestLoadConfig_PrefersFlagsOverEnv(t *testing.T) {
	getenv := func(key string) string {
		values := map[string]string{
			"SLURP_PORT":   "9001",
			"PORT":         "9002",
			"SLURP_DIR":    "/tmp/env-slurp",
			"UPLOAD_DIR":   "/tmp/env-legacy",
			"SLURP_TOKEN":  "namespaced-token",
			"UPLOAD_TOKEN": "legacy-token",
		}
		return values[key]
	}

	cfg, err := loadConfig(
		[]string{"--port", "9100", "--dir", "/tmp/flag-dir", "--token", "flag-token", "--no-tunnel"},
		getenv,
		func() (string, error) { return "/home/tester", nil },
	)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if cfg.Port != 9100 {
		t.Fatalf("expected flag port to win, got %d", cfg.Port)
	}
	if cfg.Dir != "/tmp/flag-dir" {
		t.Fatalf("expected flag dir to win, got %q", cfg.Dir)
	}
	if cfg.Token != "flag-token" {
		t.Fatalf("expected flag token to win, got %q", cfg.Token)
	}
	if !cfg.NoTunnel {
		t.Fatal("expected --no-tunnel to be applied")
	}
}

func TestLoadConfig_PrefersNamespacedEnvOverLegacy(t *testing.T) {
	getenv := func(key string) string {
		values := map[string]string{
			"SLURP_PORT":   "9001",
			"PORT":         "9002",
			"SLURP_DIR":    "/tmp/env-slurp",
			"UPLOAD_DIR":   "/tmp/env-legacy",
			"SLURP_TOKEN":  "namespaced-token",
			"UPLOAD_TOKEN": "legacy-token",
		}
		return values[key]
	}

	cfg, err := loadConfig(nil, getenv, func() (string, error) { return "/home/tester", nil })
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if cfg.Port != 9001 {
		t.Fatalf("expected namespaced env port, got %d", cfg.Port)
	}
	if cfg.Dir != "/tmp/env-slurp" {
		t.Fatalf("expected namespaced env dir, got %q", cfg.Dir)
	}
	if cfg.Token != "namespaced-token" {
		t.Fatalf("expected namespaced env token, got %q", cfg.Token)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	cfg, err := loadConfig(nil, func(string) string { return "" }, func() (string, error) {
		return "/home/tester", nil
	})
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if cfg.Port != 0 {
		t.Fatalf("expected auto-port marker 0, got %d", cfg.Port)
	}
	wantDir := filepath.Join("/home/tester", "Downloads", "slurp")
	if cfg.Dir != wantDir {
		t.Fatalf("expected default dir %q, got %q", wantDir, cfg.Dir)
	}
	if cfg.Token == "" {
		t.Fatal("expected generated token")
	}
	if cfg.NoTunnel {
		t.Fatal("expected tunnel to be enabled by default")
	}
}

func TestLoadConfig_HomeDirFailure(t *testing.T) {
	_, err := loadConfig(nil, func(string) string { return "" }, func() (string, error) {
		return "", errors.New("home missing")
	})
	if err == nil {
		t.Fatal("expected loadConfig to fail when home dir cannot be resolved")
	}
}

func TestLoadConfig_InvalidPortFromEnv(t *testing.T) {
	getenv := func(key string) string {
		if key == "SLURP_PORT" {
			return "not-a-number"
		}
		return ""
	}

	_, err := loadConfig(nil, getenv, func() (string, error) { return "/home/tester", nil })
	if err == nil {
		t.Fatal("expected invalid env port to return an error")
	}
}
