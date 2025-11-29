//go:build plugins

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"wikilite/internal/plugin"
	"wikilite/pkg/models"

	"github.com/danielgtaylor/huma/v2"
)

// PluginActionInput defines the input for a plugin action.
type PluginActionInput struct {
	Body     map[string]any `doc:"JSON payload for the action" json:"body"`
	PluginID string         `doc:"The unique ID of the plugin"             path:"pluginID"`
	Action   string         `doc:"The action to perform"                   path:"action"`
	Slug     string         `doc:"Optional article slug"                                   query:"slug" required:"false"`
}

// PluginActionOutput defines the output from a plugin action.
type PluginActionOutput struct {
	Body any `json:"body"`
}

// executePlugins executes all plugins for a given hook.
func executePlugins(
	ctx context.Context,
	pluginMgr *plugin.Manager,
	hook string,
	data string,
	context map[string]any,
	logger models.Logger,
) (string, error) {
	finalBody, pluginErrs, err := pluginMgr.ExecutePipeline(hook, data, context)
	if err != nil {
		return "", err
	}

	for _, err := range pluginErrs {
		_ = logger(
			ctx,
			models.LevelError,
			"plugin",
			fmt.Sprintf("Error executing plugin: %v", err),
			"",
		)
	}

	return finalBody, nil
}

// registerPluginRoutes registers routes specifically for plugins to receive data.
func (s *Server) registerPluginRoutes(pluginPath, pluginStoragePath, jsPkgsPath string) error {
	if pluginStoragePath == "" {
		pluginStoragePath = "plugin_storage"
	}

	pluginManger, err := plugin.NewManager(pluginStoragePath, pluginPath, jsPkgsPath)
	if err != nil {
		return fmt.Errorf("failed to initialize plugin manager: %w", err)
	}

	s.PluginManager = pluginManger

	huma.Register(s.api, huma.Operation{
		OperationID: "execute-plugin-action",
		Method:      http.MethodPost,
		Path:        "/api/plugin/{pluginID}/{action}",
		Summary:     "Execute Plugin Action",
		Description: "Trigger a specific action within a plugin, passing a JSON payload.",
		Tags:        []string{"Plugins"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handlePluginAction)

	return nil
}

// handlePluginAction bridges HTTP requests to the plugin JS runtime.
func (s *Server) handlePluginAction(
	ctx context.Context,
	input *PluginActionInput,
) (*PluginActionOutput, error) {
	user := getUserFromContext(ctx)

	payloadBytes, err := json.Marshal(input.Body)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid JSON body")
	}
	payload := string(payloadBytes)

	pluginCtx := map[string]any{
		"User": user,
		"Slug": input.Slug,
	}

	responseJSON, err := s.PluginManager.ExecutePluginAction(
		input.PluginID,
		input.Action,
		payload,
		pluginCtx,
	)
	if err != nil {
		_ = s.db.CreateLogEntry(
			ctx,
			models.LevelError,
			"plugin-action",
			err.Error(),
			input.PluginID,
		)
		return nil, huma.Error500InternalServerError(
			fmt.Sprintf("Plugin execution error: %v", err),
		)
	}

	var result any
	err = json.Unmarshal([]byte(responseJSON), &result)
	if err != nil {
		result = map[string]string{"raw": responseJSON}
	}

	resp := &PluginActionOutput{}
	resp.Body = result

	return resp, nil
}

// hasActivePlugins checks if the server has an active plugin manager with plugins.
func (s *Server) hasActivePlugins() bool {
	return s.PluginManager != nil && s.PluginManager.HasPlugins()
}
