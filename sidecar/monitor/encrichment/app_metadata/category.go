package app_metadata

import "strings"

// Normalized category codes shared across platforms. Platform-native taxonomies
// (Apple UTIs, freedesktop menu categories, winget tags) all map onto these.
const (
	CategoryDevelopment    = "development"
	CategoryProductivity   = "productivity"
	CategoryCommunication  = "communication"
	CategoryWebBrowsing    = "web-browsing"
	CategoryMedia          = "media"
	CategoryGraphicsDesign = "graphics-design"
	CategoryGames          = "games"
	CategoryUtilities      = "utilities"
	CategoryBusiness       = "business-finance"
	CategoryEducation      = "education"
	CategorySocial         = "social"
	CategorySystem         = "system"
	CategoryOther          = "other"
)

// categoryLabels gives the human-readable label for each normalized code.
var categoryLabels = map[string]string{
	CategoryDevelopment:    "Development",
	CategoryProductivity:   "Productivity",
	CategoryCommunication:  "Communication",
	CategoryWebBrowsing:    "Web Browsing",
	CategoryMedia:          "Media & Entertainment",
	CategoryGraphicsDesign: "Graphics & Design",
	CategoryGames:          "Games",
	CategoryUtilities:      "Utilities",
	CategoryBusiness:       "Business & Finance",
	CategoryEducation:      "Education",
	CategorySocial:         "Social Networking",
	CategorySystem:         "System",
	CategoryOther:          "Other",
}

// CategoryLabel returns the display label for a normalized code, or "" if the
// code is unknown/empty.
func CategoryLabel(code string) string {
	return categoryLabels[strings.TrimSpace(code)]
}

// appleCategoryCodes maps Apple's LSApplicationCategoryType UTIs to our codes.
// Keys are stored without the "public.app-category." prefix (stripped before
// lookup). See https://developer.apple.com/documentation/bundleresources/information-property-list/lsapplicationcategorytype
var appleCategoryCodes = map[string]string{
	"developer-tools":   CategoryDevelopment,
	"productivity":      CategoryProductivity,
	"business":          CategoryBusiness,
	"finance":           CategoryBusiness,
	"graphics-design":   CategoryGraphicsDesign,
	"photography":       CategoryGraphicsDesign,
	"video":             CategoryMedia,
	"music":             CategoryMedia,
	"entertainment":     CategoryMedia,
	"utilities":         CategoryUtilities,
	"social-networking": CategorySocial,
	"education":         CategoryEducation,
	"reference":         CategoryEducation,
	"medical":           CategoryOther,
	"news":              CategoryOther,
	"lifestyle":         CategoryOther,
	"travel":            CategoryOther,
	"weather":           CategoryOther,
	"sports":            CategoryOther,
	"healthcare-fitness": CategoryOther,
	"food-and-drink":    CategoryOther,
	"shopping":          CategoryOther,
	"navigation":        CategoryOther,
	"books":             CategoryEducation,
}

// CategoryFromApple maps an LSApplicationCategoryType value to a normalized
// (code, label). Handles game sub-categories (public.app-category.*-games) and
// the generic "games" bucket. Returns ("", "") when unmapped.
func CategoryFromApple(uti string) (code string, label string) {
	uti = strings.ToLower(strings.TrimSpace(uti))
	if uti == "" {
		return "", ""
	}
	uti = strings.TrimPrefix(uti, "public.app-category.")
	if uti == "games" || strings.HasSuffix(uti, "-games") {
		return CategoryGames, categoryLabels[CategoryGames]
	}
	if c, ok := appleCategoryCodes[uti]; ok {
		return c, categoryLabels[c]
	}
	return "", ""
}

// freedesktopMainCategories maps freedesktop.org "main" desktop menu categories
// to our codes. See https://specifications.freedesktop.org/menu-spec/latest/apas02.html
var freedesktopMainCategories = map[string]string{
	"development": CategoryDevelopment,
	"office":      CategoryProductivity,
	"audiovideo":  CategoryMedia,
	"audio":       CategoryMedia,
	"video":       CategoryMedia,
	"graphics":    CategoryGraphicsDesign,
	"game":        CategoryGames,
	"education":   CategoryEducation,
	"science":     CategoryEducation,
	"utility":     CategoryUtilities,
	"system":      CategorySystem,
	"settings":    CategorySystem,
	"network":     CategoryWebBrowsing,
}

// freedesktopExtraCategories are additional (non-main) tokens that refine a
// Network entry or otherwise carry a strong signal; checked before the main
// table so e.g. Network;InstantMessaging resolves to communication.
var freedesktopExtraCategories = map[string]string{
	"webbrowser":      CategoryWebBrowsing,
	"instantmessaging": CategoryCommunication,
	"chat":            CategoryCommunication,
	"email":           CategoryCommunication,
	"telephony":       CategoryCommunication,
	"videoconference": CategoryCommunication,
	"news":            CategoryOther,
	"ide":             CategoryDevelopment,
}

// CategoryFromFreedesktop maps a `Categories=` value (semicolon-separated
// tokens) to a normalized (code, label). Refining tokens (WebBrowser, Email…)
// take priority so Network entries resolve to something more specific than
// "web-browsing". Returns ("", "") when no token maps.
func CategoryFromFreedesktop(categories string) (code string, label string) {
	tokens := strings.Split(categories, ";")
	// First pass: refining/extra tokens win.
	for _, token := range tokens {
		key := strings.ToLower(strings.TrimSpace(token))
		if key == "" {
			continue
		}
		if c, ok := freedesktopExtraCategories[key]; ok {
			return c, categoryLabels[c]
		}
	}
	// Second pass: main categories.
	for _, token := range tokens {
		key := strings.ToLower(strings.TrimSpace(token))
		if key == "" {
			continue
		}
		if c, ok := freedesktopMainCategories[key]; ok {
			return c, categoryLabels[c]
		}
	}
	return "", ""
}
