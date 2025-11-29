package api

import (
	"context"
	"net/http"
	"wikilite/pkg/models"

	"github.com/danielgtaylor/huma/v2"
)

// LogsPaginationInput represents the input for paginating logs.
type LogsPaginationInput struct {
	Level models.LogLevel `doc:"Filter by log level (INFO, ERROR, etc.)" query:"level" required:"false"`
	Page  int             `doc:"Page number"                             query:"page"                   default:"1"  minimum:"1"`
	Limit int             `doc:"Items per page"                          query:"limit"                  default:"50" minimum:"1" maximum:"100"`
}

// LogsListOutput represents the output for a list of logs.
type LogsListOutput struct {
	Body struct {
		Logs  []*models.SystemLog `json:"logs"`
		Total int64               `json:"total"`
		Page  int                 `json:"page"`
		Limit int                 `json:"limit"`
	}
}

// registerLogRoutes registers the log routes with the API.
func (s *Server) registerLogRoutes() {
	huma.Register(s.api, huma.Operation{
		OperationID: "get-logs",
		Method:      http.MethodGet,
		Path:        "/api/logs",
		Summary:     "Get System Logs",
		Description: "Retrieve paginated system logs. Admin only.",
		Tags:        []string{"System"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleGetLogs)
}

// handleGetLogs handles the request to get system logs.
func (s *Server) handleGetLogs(
	ctx context.Context,
	input *LogsPaginationInput,
) (*LogsListOutput, error) {
	user := getAdminUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error403Forbidden("Only admins can view system logs")
	}

	if input.Page < 1 {
		input.Page = 1
	}

	if input.Limit < 1 {
		input.Limit = 50
	}

	offset := (input.Page - 1) * input.Limit

	logs, total, err := s.db.GetLogs(ctx, input.Limit, offset, input.Level)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	resp := &LogsListOutput{}
	resp.Body.Logs = logs
	resp.Body.Total = total
	resp.Body.Page = input.Page
	resp.Body.Limit = input.Limit

	return resp, nil
}
