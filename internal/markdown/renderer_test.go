package markdown

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRenderer(t *testing.T) {
	renderer := NewRenderer()
	require.NotNil(t, renderer)
	require.NotNil(t, renderer.md)
	require.NotNil(t, renderer.sanitizer)
}

func TestRenderer_RenderHTML_BasicMarkdown(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "# Hello World\n\nThis is a **test**."
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<h1")
	assert.Contains(t, result, "Hello World")
	assert.Contains(t, result, "<strong>test</strong>")
}

func TestRenderer_RenderHTML_GitHubFlavoredMarkdown(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "## Task List\n\n- [x] Completed task\n- [ ] Pending task"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<h2")
	assert.Contains(t, result, "Task List")
	assert.Contains(t, result, "<ul>")
	assert.Contains(t, result, "Completed task")
	assert.Contains(t, result, "Pending task")
}

func TestRenderer_RenderHTML_CodeBlocks(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "```go\nfunc hello() {\n    fmt.Println(\"Hello\")\n}\n```"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<pre><code")
	assert.Contains(t, result, "func hello()")
	assert.Contains(t, result, "fmt.Println")
	assert.Contains(t, result, "func hello()")
}

func TestRenderer_RenderHTML_Tables(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "| Name | Age |\n|------|-----|\n| John | 25  |\n| Jane | 30  |"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<table")
	assert.Contains(t, result, "<thead")
	assert.Contains(t, result, "<tbody")
	assert.Contains(t, result, "<td>John</td>")
	assert.Contains(t, result, "<td>25</td>")
}

func TestRenderer_RenderHTML_Links(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "[Google](https://google.com) and [internal link](/wiki/home)"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<a href=\"https://google.com\"")
	assert.Contains(t, result, "Google")
	assert.Contains(t, result, "<a href=\"/wiki/home\"")
	assert.Contains(t, result, "internal link")
}

func TestRenderer_RenderHTML_Images(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "![Alt text](/image.jpg \"Title\")"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<img src=\"/image.jpg\"")
	assert.Contains(t, result, "alt=\"Alt text\"")
	assert.Contains(t, result, "title=\"Title\"")
}

func TestRenderer_RenderHTML_Blockquotes(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "> This is a quote\n> > Nested quote"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<blockquote")
	assert.Contains(t, result, "This is a quote")
	assert.Contains(t, result, "<blockquote")
	assert.Contains(t, result, "Nested quote")
}

func TestRenderer_RenderHTML_HorizontalRules(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "Text above\n\n---\n\nText below"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "Text above")
	assert.Contains(t, result, "<hr")
	assert.Contains(t, result, "Text below")
}

func TestRenderer_RenderHTML_AutoHeadingIDs(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "# Test Heading With Spaces"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<h1")
	assert.Contains(t, result, "id=\"test-heading-with-spaces\"")
}

func TestRenderer_RenderHTML_Sanitization(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "<script>alert('xss')</script>\n\n# Safe Content"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.NotContains(t, result, "<script>")
	assert.NotContains(t, result, "alert('xss')")
	assert.Contains(t, result, "<h1")
	assert.Contains(t, result, "Safe Content")
}

func TestRenderer_RenderHTML_EmptyContent(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	err := renderer.RenderHTML(ctx, &buf, "")
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestRenderer_RenderHTML_OnlyWhitespace(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "   \n\n  \n  "
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)
	result := buf.String()
	assert.Empty(t, result)
}

func TestRenderer_RenderHTML_Strikethrough(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "~~strikethrough text~~"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<del>strikethrough text</del>")
}

func TestRenderer_RenderHTML_TaskLists(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "- [x] Done\n- [ ] Not done"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<ul>")
	assert.Contains(t, result, "Done")
	assert.Contains(t, result, "Not done")
}

func TestRenderer_RenderHTML_EscapedCharacters(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "\\# Not a heading\n\\*Not bold\\*"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.NotContains(t, result, "<h1")
	assert.NotContains(t, result, "<strong>")
	assert.Contains(t, result, "# Not a heading")
	assert.Contains(t, result, "*Not bold*")
}

func TestRenderer_RenderHTML_MalformedMarkdown(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	content := "# Unclosed heading\n**Bold without closing"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<h1")
	assert.Contains(t, result, "Unclosed heading")
	assert.Contains(t, result, "**Bold without closing")
}

func TestRenderer_RenderHTML_LongContent(t *testing.T) {
	renderer := NewRenderer()
	ctx := context.Background()
	var buf bytes.Buffer

	var builder strings.Builder
	builder.WriteString("# Long Document\n\n")
	for i := range 100 {
		builder.WriteString("## Section ")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteString("\n\nThis is section ")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteString(" with **bold** and *italic* text.\n\n")
	}

	err := renderer.RenderHTML(ctx, &buf, builder.String())
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "<h1")
	assert.Contains(t, result, "Long Document")
	assert.Contains(t, result, "<h2")
	assert.Contains(t, result, "Section 0")
	assert.Contains(t, result, "<strong>bold</strong>")
	assert.Contains(t, result, "<em>italic</em>")
	assert.True(t, len(result) > 1000, "Result should be substantial for long content")
}

func TestRenderer_RenderHTML_ContextCancellation(t *testing.T) {
	renderer := NewRenderer()
	ctx, cancel := context.WithCancel(context.Background())
	var buf bytes.Buffer

	cancel()

	content := "# Test"
	err := renderer.RenderHTML(ctx, &buf, content)
	require.NoError(t, err)
}
