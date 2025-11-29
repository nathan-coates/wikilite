//go:build ui

package api

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"wikilite/pkg/models"
)

//go:embed templates/*
var templateFS embed.FS

// initTemplates parses the embedded templates.
func (s *Server) initTemplates() error {
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"formatRole": func(role models.UserRole) string {
			switch role {
			case models.READ:
				return "User"
			case models.WRITE:
				return "Editor"
			case models.ADMIN:
				return "Admin"
			default:
				return "Anonymous"
			}
		},
	}

	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return fmt.Errorf("failed to read template directory: %w", err)
	}

	s.compiledTemplates = make(map[string]*template.Template)

	for _, entry := range entries {
		filename := entry.Name()

		if filename == "base.gohtml" || !strings.HasSuffix(filename, ".gohtml") {
			continue
		}

		tmpl := template.New("base.gohtml").Funcs(funcMap)

		_, err := tmpl.ParseFS(templateFS, "templates/base.gohtml", "templates/"+filename)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", filename, err)
		}

		s.compiledTemplates[filename] = tmpl
	}

	return nil
}

// templateData is the standardized structure passed to all views.
type templateData struct {
	User     *models.User
	Data     any
	WikiName string
	Error    string
	Success  string
}

// RegisterRoutes attaches all frontend-specific paths to the provided ServeMux.
func (s *Server) registerFrontendRoutes(mux *http.ServeMux) error {

	err := s.initTemplates()
	if err != nil {
		return err
	}

	if s.isExternalIDPEnabled() {
		mux.HandleFunc("GET /{path...}", s.uiRenderExternalIDPDisabled)
		return nil
	}

	// Public
	mux.HandleFunc("GET /", s.uiRenderHome)
	mux.HandleFunc("GET /wiki/{slug}", s.uiRenderArticle)
	mux.HandleFunc("GET /wiki/{slug}/history", s.uiRenderHistory)
	mux.HandleFunc("GET /wiki/{slug}/history/{version}", s.uiRenderPastVersion)

	// Auth
	mux.HandleFunc("GET /login", s.uiRenderLogin)
	mux.HandleFunc("POST /login", s.uiHandleLoginSubmit)
	mux.HandleFunc("POST /logout", s.uiHandleLogout)

	// App
	mux.HandleFunc("GET /dashboard", s.uiRenderDashboard)
	mux.HandleFunc("GET /new", s.uiRenderNewArticle)
	mux.HandleFunc("POST /new", s.uiActionCreateIntent)
	mux.HandleFunc("POST /wiki/{slug}/edit", s.uiActionEditIntent)
	mux.HandleFunc("GET /editor/{draftID}", s.uiRenderEditor)

	// Editor Actions
	mux.HandleFunc("POST /editor/{draftID}/save", s.uiActionSaveDraft)
	mux.HandleFunc("POST /editor/{draftID}/publish", s.uiActionPublishDraft)
	mux.HandleFunc("POST /editor/{draftID}/discard", s.uiActionDiscardDraft)

	// User
	mux.HandleFunc("GET /user", s.uiRenderUser)
	mux.HandleFunc("POST /user", s.uiActionUpdateUserPassword)

	// Admin Actions
	mux.HandleFunc("POST /wiki/{slug}/delete", s.uiActionDeleteArticle)
	mux.HandleFunc("GET /admin/logs", s.uiRenderLogs)

	// Special
	mux.HandleFunc("GET /special/orphans", s.uiRenderOrphans)

	return nil
}
