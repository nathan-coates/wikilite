package api

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jellydator/ttlcache/v3"
)

func (s *Server) getRenderedHTML(ctx context.Context, article *PublicArticle) (string, error) {
	key := fmt.Sprintf("%d-%d", article.Id, article.Version)

	item := s.htmlCache.Get(key)
	if item != nil {
		return item.Value(), nil
	}

	var buf bytes.Buffer

	err := s.renderer.RenderHTML(ctx, &buf, article.Data)
	if err != nil {
		return "", err
	}

	htmlContent := buf.String()

	s.htmlCache.Set(key, htmlContent, ttlcache.DefaultTTL)

	return htmlContent, nil
}
