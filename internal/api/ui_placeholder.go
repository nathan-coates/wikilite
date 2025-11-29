//go:build !ui

package api

import "net/http"

// registerFrontendRoutes is a placeholder for when the UI is not built.
func (s *Server) registerFrontendRoutes(mux *http.ServeMux) error {
	return nil
}
