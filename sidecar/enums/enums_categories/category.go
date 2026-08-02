//nolint:goconst // some category codes repeat but it is false duplication
package enumscategories

import (
	"fmt"
	"strings"
)

type Category int

const (
	CategoryUnknown Category = iota
	CategoryDevelopment
	CategoryProductivity
	CategoryCommunication
	CategoryWebBrowsing
	CategoryMedia
	CategoryGraphicsDesign
	CategoryGames
	CategoryUtilities
	CategoryBusiness
	CategoryEducation
	CategorySocial
	CategorySystem
	CategoryOther
)

func (category Category) String() string {
	categories := []string{
		"unknown",
		"development",
		"productivity",
		"communication",
		"web-browsing",
		"media",
		"graphics-design",
		"games",
		"utilities",
		"business-finance",
		"education",
		"social",
		"system",
		"other",
	}

	return categories[category]
}

func (category Category) Label() string {
	labels := []string{
		"Unknown",
		"Development",
		"Productivity",
		"Communication",
		"Web Browsing",
		"Media & Entertainment",
		"Graphics & Design",
		"Games",
		"Utilities",
		"Business & Finance",
		"Education",
		"Social Networking",
		"System",
		"Other",
	}

	return labels[category]
}

func (category Category) IsProductive() bool {
	switch category {
	case CategoryDevelopment, CategoryProductivity, CategoryGraphicsDesign, CategoryBusiness,
		CategoryEducation:
		return true
	case CategoryUnknown, CategoryCommunication, CategoryWebBrowsing, CategoryMedia, CategoryGames,
		CategoryUtilities, CategorySocial, CategorySystem, CategoryOther:
		return false
	}

	return false
}

// Parse is the inverse of String: every code the taxonomy emits round-trips back to its
// category. The seeded `application_categories` rows are exactly those codes, so a read that
// resolves a persisted classification by code depends on this staying total.
func Parse(code string) (Category, error) {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "unknown":
		return CategoryUnknown, nil
	case "development":
		return CategoryDevelopment, nil
	case "productivity":
		return CategoryProductivity, nil
	case "communication":
		return CategoryCommunication, nil
	case "web-browsing":
		return CategoryWebBrowsing, nil
	case "media":
		return CategoryMedia, nil
	case "graphics-design":
		return CategoryGraphicsDesign, nil
	case "games":
		return CategoryGames, nil
	case "utilities":
		return CategoryUtilities, nil
	case "business-finance":
		return CategoryBusiness, nil
	case "education":
		return CategoryEducation, nil
	case "social":
		return CategorySocial, nil
	case "system":
		return CategorySystem, nil
	case "other":
		return CategoryOther, nil
	}

	return CategoryUnknown, fmt.Errorf("unrecognized category: %s", code)
}

// ParseAppleCategory reads an LSApplicationCategoryType code. Codes carry the
// "public.app-category." prefix, and any "*-games" subcategory folds into games.
// See https://developer.apple.com/documentation/bundleresources/information-property-list/lsapplicationcategorytype
func ParseAppleCategory(code string) (Category, error) {
	switch strings.TrimPrefix(strings.ToLower(strings.TrimSpace(code)), "public.app-category.") {
	case "developer-tools":
		return CategoryDevelopment, nil
	case "productivity":
		return CategoryProductivity, nil
	case "business", "finance":
		return CategoryBusiness, nil
	case "graphics-design", "photography":
		return CategoryGraphicsDesign, nil
	case "video", "music", "entertainment":
		return CategoryMedia, nil
	case "utilities":
		return CategoryUtilities, nil
	case "social-networking":
		return CategorySocial, nil
	case "education", "reference", "books":
		return CategoryEducation, nil
	case "medical", "news", "lifestyle", "travel", "weather", "sports",
		"healthcare-fitness", "food-and-drink", "shopping", "navigation":
		return CategoryOther, nil
	case "games":
		return CategoryGames, nil
	default:
		if strings.HasSuffix(code, "-games") {
			return CategoryGames, nil
		}
	}

	return CategoryUnknown, fmt.Errorf("unrecognized category: %s", code)
}

// ParseFreedesktopCategory see: https://specifications.freedesktop.org/menu-spec/latest/apas02.html
func ParseFreedesktopCategory(code string) (Category, error) {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "webbrowser":
		return CategoryWebBrowsing, nil
	case "instantmessaging", "chat", "email", "telephony", "videoconference":
		return CategoryCommunication, nil
	case "news":
		return CategoryOther, nil
	case "ide":
		return CategoryDevelopment, nil
	case "development":
		return CategoryDevelopment, nil
	case "office":
		return CategoryProductivity, nil
	case "audiovideo", "audio", "video":
		return CategoryMedia, nil
	case "graphics":
		return CategoryGraphicsDesign, nil
	case "game":
		return CategoryGames, nil
	case "education", "science":
		return CategoryEducation, nil
	case "utility":
		return CategoryUtilities, nil
	case "system", "settings":
		return CategorySystem, nil
	case "network":
		return CategoryWebBrowsing, nil
	}

	return CategoryUnknown, fmt.Errorf("unrecognized category: %s", code)
}

func ParseFreedesktopCategories(categories string) (Category, error) {
	parsed := []Category{}

	for code := range strings.SplitSeq(categories, ";") {
		if code, err := ParseFreedesktopCategory(code); err == nil {
			parsed = append(parsed, code)
		}
	}

	if len(parsed) == 0 {
		return CategoryUnknown, fmt.Errorf("unrecognized categories: %v", categories)
	}

	return parsed[len(parsed)-1], nil
}

// ParseWingetKeyword matches one winget keyword. Keywords are free text, so a miss is expected.
func ParseWingetKeyword(keyword string) (Category, error) {
	switch strings.ToLower(strings.TrimSpace(keyword)) {
	case "developer", "development", "ide", "editor", "terminal", "git":
		return CategoryDevelopment, nil
	case "browser":
		return CategoryWebBrowsing, nil
	case "chat", "messaging", "email":
		return CategoryCommunication, nil
	case "video", "audio", "music", "media", "player":
		return CategoryMedia, nil
	case "game", "gaming":
		return CategoryGames, nil
	case "design", "graphics", "photo":
		return CategoryGraphicsDesign, nil
	case "office", "productivity", "note":
		return CategoryProductivity, nil
	case "utility", "utilities":
		return CategoryUtilities, nil
	case "social":
		return CategorySocial, nil
	case "education":
		return CategoryEducation, nil
	case "finance", "business":
		return CategoryBusiness, nil
	}

	return CategoryUnknown, fmt.Errorf("unrecognized keyword: %s", keyword)
}

func ParseWingetTags(tags []string) (Category, error) {
	for _, tag := range tags {
		if category, err := ParseWingetKeyword(tag); err == nil {
			return category, nil
		}
	}

	return CategoryUnknown, fmt.Errorf("unrecognized tags: %v", tags)
}
