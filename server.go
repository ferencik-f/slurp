package main

import (
	"net/http"
	"strings"
)

func checkAuth(r *http.Request, token string) bool {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ") == token
	}
	return r.URL.Query().Get("token") == token
}
