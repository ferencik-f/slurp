package main

import "testing"

func TestGenerateToken(t *testing.T) {
	tok, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken returned error: %v", err)
	}
	if len(tok) != 32 {
		t.Fatalf("expected 32 hex chars, got %d: %q", len(tok), tok)
	}
	tok2, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken returned error: %v", err)
	}
	if tok == tok2 {
		t.Fatal("two tokens should not be equal")
	}
}
