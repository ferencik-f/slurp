package main

import (
	"net/http"
	"testing"
)

func TestCheckAuth_BearerHeader(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload", nil)
	r.Header.Set("Authorization", "Bearer secret123")
	if !checkAuth(r, "secret123") {
		t.Fatal("expected auth to pass with correct Bearer token")
	}
}

func TestCheckAuth_QueryParam(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload?token=secret123", nil)
	if !checkAuth(r, "secret123") {
		t.Fatal("expected auth to pass with correct query token")
	}
}

func TestCheckAuth_WrongToken(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload?token=wrong", nil)
	if checkAuth(r, "secret123") {
		t.Fatal("expected auth to fail with wrong token")
	}
}

func TestCheckAuth_Missing(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload", nil)
	if checkAuth(r, "secret123") {
		t.Fatal("expected auth to fail with no token")
	}
}

func TestCheckAuth_HeaderTakesPrecedence(t *testing.T) {
	r, _ := http.NewRequest("PUT", "/upload?token=wrong", nil)
	r.Header.Set("Authorization", "Bearer secret123")
	if !checkAuth(r, "secret123") {
		t.Fatal("header should take precedence over query param")
	}
}