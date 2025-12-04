//go:build ui

package api

import (
	"net/http"
	"strings"

	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/unrolled/secure"
)

const baseRateLimiter = 30

// hardeningMiddleware returns a middleware chain that applies security hardening to the HTTP routes.
func (s *Server) hardeningMiddleware(next http.Handler) http.Handler {
	sslProxyHeaders := make(map[string]string)
	ipLookups := make([]limiter.IPLookup, 0)

	if s.trustProxyHeaders {
		sslProxyHeaders = map[string]string{"X-Forwarded-Proto": "https"}
		ipLookups = []limiter.IPLookup{
			{
				Name:           "RemoteAddr",
				IndexFromRight: 0,
			},
			{
				Name:           "X-Forwarded-For",
				IndexFromRight: 0,
			},
			{
				Name:           "X-Real-IP",
				IndexFromRight: 0,
			},
		}
	}

	secureMiddleware := secure.New(secure.Options{
		STSSeconds:           31536000,
		STSIncludeSubdomains: true,
		FrameDeny:            true,
		ContentTypeNosniff:   true,
		BrowserXssFilter:     true,
		IsDevelopment:        !s.production,
		SSLProxyHeaders:      sslProxyHeaders,
	})

	var rateLimitMiddleware func(next http.Handler) http.Handler

	if s.production {
		rateLimiter := tollbooth.NewLimiter(baseRateLimiter, nil)
		for _, lookup := range ipLookups {
			rateLimiter.SetIPLookup(lookup)
		}

		rateLimitMiddleware = func(next http.Handler) http.Handler {
			return tollbooth.LimitHandler(rateLimiter, next)
		}
	} else {
		rateLimitMiddleware = func(next http.Handler) http.Handler {
			return next
		}
	}

	csrfMiddleware := http.CrossOriginProtection{}

	conditionalCSRF := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/docs") {
			next.ServeHTTP(w, r)
			return
		}

		csrfMiddleware.Handler(next).ServeHTTP(w, r)
	})

	return rateLimitMiddleware(secureMiddleware.Handler(conditionalCSRF))
}
