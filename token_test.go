package main

import "testing"

func TestGenerateToken(t *testing.T) {
	tok := generateToken()
	if len(tok) != 16 {
		t.Fatalf("expected 16 hex chars, got %d: %q", len(tok), tok)
	}
	tok2 := generateToken()
	if tok == tok2 {
		t.Fatal("two tokens should not be equal")
	}
}
