package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port     int
	Dir      string
	Token    string
	NoTunnel bool
}

func loadConfig(args []string, getenv func(string) string, userHomeDir func() (string, error)) (Config, error) {
	fs := flag.NewFlagSet("slurp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	portFlag := fs.Int("port", 0, "listen port (default: first free from 8765)")
	dirFlag := fs.String("dir", "", "upload directory (default: ~/Downloads/slurp)")
	tokenFlag := fs.String("token", "", "auth token (default: auto-generated)")
	noTunnelFlag := fs.Bool("no-tunnel", false, "disable cloudflared tunnel")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	port, err := configuredPort(*portFlag, getenv)
	if err != nil {
		return Config{}, err
	}

	dir, err := configuredDir(*dirFlag, getenv, userHomeDir)
	if err != nil {
		return Config{}, err
	}

	token, err := configuredToken(*tokenFlag, getenv)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:     port,
		Dir:      dir,
		Token:    token,
		NoTunnel: *noTunnelFlag,
	}, nil
}

func configuredPort(flagValue int, getenv func(string) string) (int, error) {
	if flagValue != 0 {
		return flagValue, nil
	}

	value := firstEnv(getenv, "SLURP_PORT", "PORT")
	if value == "" {
		return 0, nil
	}

	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q: %w", value, err)
	}
	if port < 0 || port > 65535 {
		return 0, fmt.Errorf("port %d out of range", port)
	}
	return port, nil
}

func configuredDir(flagValue string, getenv func(string) string, userHomeDir func() (string, error)) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if value := firstEnv(getenv, "SLURP_DIR", "UPLOAD_DIR"); value != "" {
		return value, nil
	}

	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, "Downloads", "slurp"), nil
}

func configuredToken(flagValue string, getenv func(string) string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if value := firstEnv(getenv, "SLURP_TOKEN", "UPLOAD_TOKEN"); value != "" {
		return value, nil
	}

	token, err := generateToken()
	if err != nil {
		return "", err
	}
	return token, nil
}

func firstEnv(getenv func(string) string, keys ...string) string {
	for _, key := range keys {
		if value := getenv(key); value != "" {
			return value
		}
	}
	return ""
}
