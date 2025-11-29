package api

import (
	"fmt"
	"html/template"

	"github.com/danielgtaylor/huma/v2"
	"gopkg.in/yaml.v3"
)

// streamHTML streams the HTML representation of an article using Server dependencies.
func (s *Server) streamHTML(article *PublicArticle) *huma.StreamResponse {
	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
			w := ctx.BodyWriter()

			wikiContent, err := s.getRenderedHTML(ctx.Context(), article)
			if err != nil {
				_, _ = fmt.Fprintf(w, "\n%v", err)

				return
			}

			var author string
			if article.Author != nil {
				author = *article.Author
			}

			if s.hasActivePlugins() {
				pluginCtx := map[string]any{
					"User": getUserFromContext(ctx.Context()),
					"Slug": article.Slug,
				}

				finalBody, err := executePlugins(
					ctx.Context(),
					s.PluginManager,
					"onArticleRender",
					wikiContent,
					pluginCtx,
					s.db.CreateLogEntry,
				)
				if err != nil {
					_, _ = fmt.Fprintf(w, "\n<!-- Error executing plugins: %v -->", err)
					return
				}

				wikiContent = finalBody
			}

			data := struct {
				Title   string
				Author  string
				Content template.HTML
				Id      int
				Version int
			}{
				Id:      article.Id,
				Version: article.Version,
				Title:   article.Title,
				Author:  author,
				Content: template.HTML(wikiContent),
			}

			err = s.articleTemplate.Execute(w, data)
			if err != nil {
				_, _ = fmt.Fprintf(w, "\n%v", err)
			}
		},
	}
}

// streamMarkdown constructs a file with Frontmatter + Content.
func (s *Server) streamMarkdown(article *PublicArticle) *huma.StreamResponse {
	metadata := map[string]any{
		"id":      article.Id,
		"title":   article.Title,
		"slug":    article.Slug,
		"version": article.Version,
		"date":    article.CreatedAt,
	}

	if article.Author != nil {
		metadata["author"] = *article.Author
	}

	fm, _ := yaml.Marshal(metadata)

	fullDoc := fmt.Sprintf("---\n%s---\n\n%s", string(fm), article.Data)

	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Content-Type", "text/markdown; charset=utf-8")
			w := ctx.BodyWriter()
			_, _ = w.Write([]byte(fullDoc))
		},
	}
}
