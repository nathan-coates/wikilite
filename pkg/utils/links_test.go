package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToKebabCase_BasicConversion(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Test String", "test-string"},
		{"UPPERCASE", "uppercase"},
		{"lowercase", "lowercase"},
		{"Mixed CASE String", "mixed-case-string"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ToKebabCase(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToKebabCase_SpecialCharacters(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Hello@World!", "hello-world"},
		{"Test#String$With%Special^Chars", "test-string-with-special-chars"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Tabs\tAnd\nNewlines", "tabs-and-newlines"},
		{"Punctuation...right?yes!", "punctuation-right-yes"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ToKebabCase(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToKebabCase_Numbers(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Test123", "test123"},
		{"123 Numbers", "123-numbers"},
		{"Version 2.0", "version-2-0"},
		{"Test 1 2 3", "test-1-2-3"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ToKebabCase(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToKebabCase_EdgeCases(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"   ", ""},
		{"---", ""},
		{"!!!", ""},
		{"-Leading", "leading"},
		{"Trailing-", "trailing"},
		{"-Both-", "both"},
		{"---Multiple---Hyphens---", "multiple-hyphens"},
	}

	for _, tc := range testCases {
		t.Run("edge_"+tc.input, func(t *testing.T) {
			result := ToKebabCase(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToKebabCase_AlreadyKebabCase(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"already-kebab-case", "already-kebab-case"},
		{"test-string", "test-string"},
		{"123-numbers", "123-numbers"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ToKebabCase(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToKebabCase_SnakeCase(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"snake_case_string", "snake-case-string"},
		{"test_case", "test-case"},
		{"already_snake_case", "already-snake-case"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ToKebabCase(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToKebabCase_CamelCase(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"camelCaseString", "camelcasestring"},
		{"PascalCaseString", "pascalcasestring"},
		{"XMLHttpRequest", "xmlhttprequest"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ToKebabCase(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractSlugsFromContent_BasicLinks(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "single internal link",
			content:  "[Home](/wiki/home)",
			expected: []string{"home"},
		},
		{
			name:     "multiple internal links",
			content:  "[Home](/wiki/home) and [About](/wiki/about)",
			expected: []string{"home", "about"},
		},
		{
			name:     "internal and external links",
			content:  "[Home](/wiki/home) and [Google](https://google.com)",
			expected: []string{"home"},
		},
		{
			name:     "only external links",
			content:  "[Google](https://google.com) and [GitHub](https://github.com)",
			expected: []string{},
		},
		{
			name:     "links without /wiki/ prefix",
			content:  "[Home](/home) and [About](/about)",
			expected: []string{"home", "about"},
		},
		{
			name:     "mixed link formats",
			content:  "[Home](/wiki/home) [External](https://example.com) [Simple](/simple)",
			expected: []string{"home", "simple"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractSlugsFromContent(tc.content)
			assert.ElementsMatch(t, tc.expected, result)
		})
	}
}

func TestExtractSlugsFromContent_DuplicateLinks(t *testing.T) {
	content := "[Home](/wiki/home) and [Home](/wiki/home) again"
	result := ExtractSlugsFromContent(content)
	assert.Equal(t, []string{"home"}, result)
}

func TestExtractSlugsFromContent_ComplexMarkdown(t *testing.T) {
	content := `# Title

This is a paragraph with [Home](/wiki/home) link.

## Lists
- [Item 1](/wiki/item1)
- [Item 2](https://external.com)
- [Item 3](/wiki/item3)

` + "`" + `[Code](/wiki/not-a-link)` + "`" + ` should not be extracted.

[Final Link](/wiki/final)`

	result := ExtractSlugsFromContent(content)
	expected := []string{"home", "item1", "item3", "final", "not-a-link"}
	assert.ElementsMatch(t, expected, result)
}

func TestExtractSlugsFromContent_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: []string{},
		},
		{
			name:     "no links",
			content:  "Just plain text without any links",
			expected: []string{},
		},
		{
			name:     "malformed links",
			content:  "[Broken link](/wiki/broken and [Another](missing",
			expected: []string{},
		},
		{
			name:     "empty link text",
			content:  "[](/wiki/empty)",
			expected: []string{"empty"},
		},
		{
			name:     "empty link URL",
			content:  "[Empty]()",
			expected: []string{},
		},
		{
			name:     "link with only slash",
			content:  "[Root](/)",
			expected: []string{},
		},
		{
			name:     "link with trailing slash",
			content:  "[Trailing](/wiki/slug/)",
			expected: []string{"slug"},
		},
		{
			name:     "link with multiple slashes",
			content:  "[Complex](/wiki/nested/path)",
			expected: []string{"nested/path"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractSlugsFromContent(tc.content)
			assert.ElementsMatch(t, tc.expected, result)
		})
	}
}

func TestExtractSlugsFromContent_SpecialCharactersInLinks(t *testing.T) {
	content := `[Simple](/wiki/simple) and [With Spaces](/wiki/with spaces) and [With-Hyphens](/wiki/with-hyphens)`
	result := ExtractSlugsFromContent(content)
	expected := []string{"simple", "with spaces", "with-hyphens"}
	assert.ElementsMatch(t, expected, result)
}

func TestExtractSlugsFromContent_NestedBrackets(t *testing.T) {
	content := `[Link with [nested] brackets](/wiki/nested)`
	result := ExtractSlugsFromContent(content)
	expected := []string{"nested"}
	assert.ElementsMatch(t, expected, result)
}

func TestExtractSlugsFromContent_RelativeLinks(t *testing.T) {
	content := `[Relative](../relative) and [Root](/root) and [Wiki](/wiki/wiki)`
	result := ExtractSlugsFromContent(content)
	expected := []string{"../relative", "root", "wiki"}
	assert.ElementsMatch(t, expected, result)
}

func TestExtractSlugsFromContent_AnchorLinks(t *testing.T) {
	content := `[Anchor](#section) and [Wiki Anchor](/wiki/page#section)`
	result := ExtractSlugsFromContent(content)
	expected := []string{"page#section", "#section"}
	assert.ElementsMatch(t, expected, result)
}

func TestExtractSlugsFromContent_LargeContent(t *testing.T) {
	var content strings.Builder
	content.WriteString("# Large Document\n\n")
	expected := make([]string, 0, 100)

	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := 0; i < 100 && i < len(validChars); i++ {
		char := string(validChars[i])
		slug := "page-" + char
		content.WriteString("[Link " + char + "](/wiki/" + slug + ")\n")
		expected = append(expected, slug)
	}

	result := ExtractSlugsFromContent(content.String())
	assert.ElementsMatch(t, expected, result)
	assert.Len(t, result, len(expected))
}
