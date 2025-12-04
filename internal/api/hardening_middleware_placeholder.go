//go:build !ui

package api

import (
	"net/http"
)

// hardeningMiddleware is a placeholder for when the UI is not built.
func (s *Server) hardeningMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
