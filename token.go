package main

import (
	"crypto/rand"
	"encoding/hex"
)

func generateToken() string {
	b := make([]byte, 8) // 8 bytes → 16 hex chars
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
