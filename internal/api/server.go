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
	DefaultPort     = 8080
	cacheTtl        = 30 * time.Minute
	cacheSize       = 1000
)

type ServerConfig struct {
	Database          *db.DB
	JwtSecret         string
	JwksURL           string
	JwtIssuer         string
	JwtEmailClaim     string
	WikiName          string
	PluginPath        string
	PluginStoragePath string
	JsPkgsPath        string
	Production        bool
	TrustProxyHeaders bool
	InsecureCookies   bool
	Port              int
}

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
	port              int

	PluginManager *plugin.Manager

	htmlCache      *ttlcache.Cache[string, string]
	otpCache       *ttlcache.Cache[string, string]
	jwksURL        string
	externalIssuer string
	jwtEmailClaim  string

	WikiName    string
	LocalIssuer string

	jwtSecret []byte

	production        bool
	trustProxyHeaders bool
	insecureCookies   bool
}

// isExternalIDPEnabled returns true if external IDP support is configured.
func (s *Server) isExternalIDPEnabled() bool {
	return s.jwksURL != ""
}

// NewServer creates a new instance of the server.
func NewServer(
	config ServerConfig,
) (*Server, error) {
	router := http.NewServeMux()

	localIssuer := utils.ToKebabCase(config.WikiName)

	humaConfig := huma.DefaultConfig(config.WikiName+" API", "1.0.0")
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}

	api := humago.New(router, humaConfig)

	mdRenderer := markdown.NewRenderer()

	tmpl, err := template.New("article").Parse(articleTemplateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse article template: %w", err)
	}

	server := &Server{
		db:                config.Database,
		router:            router,
		api:               api,
		renderer:          mdRenderer,
		articleTemplate:   tmpl,
		jwtSecret:         []byte(config.JwtSecret),
		WikiName:          config.WikiName,
		LocalIssuer:       localIssuer,
		jwksURL:           config.JwksURL,
		externalIssuer:    config.JwtIssuer,
		jwtEmailClaim:     config.JwtEmailClaim,
		production:        config.Production,
		trustProxyHeaders: config.TrustProxyHeaders,
		insecureCookies:   config.InsecureCookies,
		port:              config.Port,
	}

	if config.JwksURL != "" {
		jwks, err := keyfunc.NewDefaultCtx(context.Background(), []string{config.JwksURL})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JWKS from %s: %w", config.JwksURL, err)
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

	if config.PluginPath != "" {
		err = server.registerPluginRoutes(
			config.PluginPath,
			config.PluginStoragePath,
			config.JsPkgsPath,
		)
		if err != nil {
			return nil, err
		}
	}

	htmlCache := ttlcache.New[string, string](
		ttlcache.WithTTL[string, string](cacheTtl),
		ttlcache.WithCapacity[string, string](cacheSize),
	)
	go htmlCache.Start()

	otpCache := ttlcache.New[string, string](
		ttlcache.WithTTL[string, string](10*time.Minute),
		ttlcache.WithCapacity[string, string](1000),
	)
	go otpCache.Start()

	server.htmlCache = htmlCache
	server.otpCache = otpCache

	return server, nil
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	handler := s.hardeningMiddleware(s.router)
	handler = s.LoggerMiddleware(handler)
	handler = s.authMiddleware(handler)
	handler = s.contextMiddleware(handler)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
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

	if s.otpCache != nil {
		s.otpCache.Stop()
	}

	if s.PluginManager != nil {
		err := s.PluginManager.Close()
		if err != nil {
			return fmt.Errorf("failed to close plugin manager: %w", err)
		}
	}

	return nil
}
