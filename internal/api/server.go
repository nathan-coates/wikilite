package api

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"
	"wikilite/internal/db"
	"wikilite/internal/markdown"
	"wikilite/internal/plugin"
	"wikilite/pkg/utils"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/jellydator/ttlcache/v3"
)

const (
	DefaultWikiName = "WikiLite"
	cacheTtl        = 30 * time.Minute
	cacheSize       = 1000
)

// Server represents the main application server.
type Server struct {
	api               huma.API
	jwks              keyfunc.Keyfunc
	db                *db.DB
	router            *http.ServeMux
	renderer          *markdown.Renderer
	articleTemplate   *template.Template
	compiledTemplates map[string]*template.Template
	httpServer        *http.Server

	PluginManager *plugin.Manager

	htmlCache      *ttlcache.Cache[string, string]
	jwksURL        string
	externalIssuer string
	jwtEmailClaim  string

	WikiName    string
	LocalIssuer string

	jwtSecret []byte
}

// NewServer creates a new instance of the server.
func NewServer(
	database *db.DB,
	jwtSecret string,
	jwksURL string,
	jwtIssuer string,
	jwtEmailClaim string,
	wikiName string,
	pluginPath string,
	pluginStoragePath string,
	jsPkgsPath string,
) (*Server, error) {
	router := http.NewServeMux()

	localIssuer := utils.ToKebabCase(wikiName)

	config := huma.DefaultConfig(wikiName+" API", "1.0.0")
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}

	api := humago.New(router, config)

	mdRenderer := markdown.NewRenderer()

	tmpl, err := template.New("article").Parse(articleTemplateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse article template: %w", err)
	}

	server := &Server{
		db:              database,
		router:          router,
		api:             api,
		renderer:        mdRenderer,
		articleTemplate: tmpl,
		jwtSecret:       []byte(jwtSecret),
		WikiName:        wikiName,
		LocalIssuer:     localIssuer,
		jwksURL:         jwksURL,
		externalIssuer:  jwtIssuer,
		jwtEmailClaim:   jwtEmailClaim,
	}

	if jwksURL != "" {
		jwks, err := keyfunc.NewDefaultCtx(context.Background(), []string{jwksURL})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JWKS from %s: %w", jwksURL, err)
		}

		server.jwks = jwks
	}

	server.registerArticleRoutes()
	server.registerUserRoutes()
	server.registerDraftRoutes()
	server.registerLogRoutes()
	server.registerAuthRoutes()

	err = server.registerFrontendRoutes(router)
	if err != nil {
		return nil, err
	}

	if pluginPath != "" {
		if pluginStoragePath == "" {
			pluginStoragePath = "plugin_storage"
		}

		pluginManger, err := plugin.NewManager(pluginStoragePath, pluginPath, jsPkgsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize plugin manager: %w", err)
		}

		server.PluginManager = pluginManger
		server.registerPluginRoutes()
	}

	htmlCache := ttlcache.New[string, string](
		ttlcache.WithTTL[string, string](cacheTtl),
		ttlcache.WithCapacity[string, string](cacheSize),
	)
	go htmlCache.Start()

	server.htmlCache = htmlCache

	return server, nil
}

// Start starts the HTTP server.
func (s *Server) Start(addr string) error {
	handler := s.LoggerMiddleware(s.router)
	handler = s.authMiddleware(handler)
	handler = s.contextMiddleware(handler)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// Close cleans up internal resources like plugins and caches.
func (s *Server) Close() error {
	if s.htmlCache != nil {
		s.htmlCache.Stop()
	}

	if s.PluginManager != nil {
		err := s.PluginManager.Close()
		if err != nil {
			return fmt.Errorf("failed to close plugin manager: %w", err)
		}
	}

	return nil
}
