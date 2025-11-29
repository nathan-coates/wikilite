package utils

import (
	"regexp"
	"strings"
)

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]+`)
	trimHyphensRegex     = regexp.MustCompile(`^-+|-+$`)
)

// ToKebabCase converts a string to kebab case.
func ToKebabCase(str string) string {
	str = strings.ToLower(str)

	str = nonAlphanumericRegex.ReplaceAllString(str, "-")

	str = trimHyphensRegex.ReplaceAllString(str, "")

	return str
}

// linkRegex is a regular expression to find Markdown links.
var linkRegex = regexp.MustCompile(`\[.*?\]\((.*?)\)`)

// ExtractSlugsFromContent is a helper to grab link targets.
func ExtractSlugsFromContent(content string) []string {
	matches := linkRegex.FindAllStringSubmatch(content, -1)
	uniqueSlugs := make(map[string]struct{})

	for _, match := range matches {
		if len(match) > 1 {
			url := match[1]
			if strings.HasPrefix(url, "http") {
				continue
			}

			slug := strings.TrimPrefix(url, "/wiki/")
			slug = strings.Trim(slug, "/")

			if slug != "" {
				uniqueSlugs[slug] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(uniqueSlugs))
	for slug := range uniqueSlugs {
		result = append(result, slug)
	}

	return result
}
