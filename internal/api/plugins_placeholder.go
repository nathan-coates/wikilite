//go:build !plugins

package api

import (
	"context"
	"wikilite/internal/plugin"
	"wikilite/pkg/models"
)

// executePlugins is a placeholder function for when the plugin system is not built.
func executePlugins(
	_ context.Context,
	_ *plugin.Manager,
	_ string,
	_ string,
	_ map[string]any,
	_ models.Logger,
) (string, error) {
	return "", nil
}

// hasActivePlugins is a placeholder method for when the plugin system is not built.
func (s *Server) hasActivePlugins() bool {
	return false
}

// registerPluginRoutes is a placeholder method for when the plugin system is not built.
func (s *Server) registerPluginRoutes(pluginPath, pluginStoragePath, jsPkgsPath string) error {
	return nil
}
