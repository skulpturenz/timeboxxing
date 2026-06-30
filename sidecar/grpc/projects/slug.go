package projects

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

var nonProjectSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

func uniqueProjectID(ctx context.Context, querier queries.Querier, name string) (string, error) {
	base := projectSlug(name)
	candidate := base
	suffix := 2
	for {
		count, err := querier.CountProjectsByID(ctx, candidate)
		if err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = base + "-" + strconv.Itoa(suffix)
		suffix += 1
	}
}

func projectSlug(name string) string {
	slug := nonProjectSlugChars.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "project"
	}
	return slug
}
