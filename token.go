package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func generateToken() (string, error) {
	b := make([]byte, 16) // 16 bytes → 32 hex chars
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
