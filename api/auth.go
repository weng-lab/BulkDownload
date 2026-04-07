package api

import (
	"net/http"
	"strings"
)

func adminAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.TrimSpace(token) == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if r.Header.Get("X-Admin-Token") != token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
