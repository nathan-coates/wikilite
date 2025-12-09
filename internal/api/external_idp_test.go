package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsExternalIDPEnabled(t *testing.T) {
	tests := []struct {
		name     string
		jwksURL  string
		expected bool
	}{
		{
			name:     "External IDP disabled",
			jwksURL:  "",
			expected: false,
		},
		{
			name:     "External IDP enabled",
			jwksURL:  "https://example.com/.well-known/jwks.json",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				jwksURL: tt.jwksURL,
			}
			assert.Equal(t, tt.expected, server.isExternalIDPEnabled())
		})
	}
}

func TestServerCreationWithExternalIDP(t *testing.T) {
	config := ServerConfig{
		Database:          nil,
		JwtSecret:         "test-secret",
		JwksURL:           "https://example.com/.well-known/jwks.json",
		JwtIssuer:         "https://example.com/",
		JwtEmailClaim:     "email",
		WikiName:          "Test Wiki",
		PluginPath:        "",
		PluginStoragePath: "",
		JsPkgsPath:        "",
		Production:        false,
		TrustProxyHeaders: false,
	}

	server, err := NewServer(config)

	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.True(t, server.isExternalIDPEnabled())
}

func TestServerCreationWithoutExternalIDP(t *testing.T) {
	config := ServerConfig{
		Database:          nil,
		JwtSecret:         "test-secret",
		JwksURL:           "",
		JwtIssuer:         "",
		JwtEmailClaim:     "",
		WikiName:          "Test Wiki",
		PluginPath:        "",
		PluginStoragePath: "",
		JsPkgsPath:        "",
		Production:        false,
		TrustProxyHeaders: false,
	}

	server, err := NewServer(config)

	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.False(t, server.isExternalIDPEnabled())
}
