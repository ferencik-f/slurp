package main

import (
	"crypto/rand"
	"encoding/hex"
)

func generateToken() string {
	b := make([]byte, 16) // 16 bytes → 32 hex chars
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
