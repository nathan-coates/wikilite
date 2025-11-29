package markdown

import (
	"bytes"
	"context"
	"io"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// Renderer handles the conversion of markdown to other formats.
type Renderer struct {
	md        goldmark.Markdown
	sanitizer *bluemonday.Policy
}

// NewRenderer creates a new instance of the Markdown Renderer.
func NewRenderer() *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	sanitizer := bluemonday.UGCPolicy()

	return &Renderer{
		md:        md,
		sanitizer: sanitizer,
	}
}

// RenderHTML converts markdown content to HTML, sanitizes it, and writes it to the writer.
func (r *Renderer) RenderHTML(ctx context.Context, w io.Writer, content string) error {
	var buf bytes.Buffer

	err := r.md.Convert([]byte(content), &buf)
	if err != nil {
		return err
	}

	safeHTML := r.sanitizer.SanitizeBytes(buf.Bytes())

	_, err = w.Write(safeHTML)

	return err
}
