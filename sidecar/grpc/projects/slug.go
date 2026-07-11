package projects

import (
	"regexp"
	"strings"
)

var nonProjectSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

func projectSlug(name string) string {
	slug := nonProjectSlugChars.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "project"
	}
	return slug
}
