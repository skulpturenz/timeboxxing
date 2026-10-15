package models

import (
	"strings"

	"github.com/negrel/assert"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	enumsmaskingcategory "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_masking_category"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

// MaskedPlaceholder is where masking fails closed: a value the tables do not carry never passes
// through in the clear.
const MaskedPlaceholder = "masked"

type MaskValue struct {
	MaskingCategory enumsmaskingcategory.MaskingCategory
	Value           string
}

// MaskingTables holds a permutation in Categories, so [enumscategories.Category.IsProductive] and
// .Label() answer for the stand-in — never derive productivity from masked data.
type MaskingTables struct {
	Values     map[MaskValue]string
	Categories map[enumscategories.Category]enumscategories.Category
	reverse    map[MaskValue]string
}

func MaskingTablesFrom(
	values map[MaskValue]string,
	categories map[enumscategories.Category]enumscategories.Category,
) MaskingTables {
	reverse := make(map[MaskValue]string, len(values))
	for input, masked := range values {
		reverse[MaskValue{MaskingCategory: input.MaskingCategory, Value: masked}] = input.Value
	}

	return MaskingTables{Values: values, Categories: categories, reverse: reverse}
}

func (tables MaskingTables) Unmask(maskingCategory enumsmaskingcategory.MaskingCategory, token string) (string, bool) {
	value, ok := tables.reverse[MaskValue{MaskingCategory: maskingCategory, Value: token}]

	return value, ok
}

func (tables MaskingTables) mask(maskingCategory enumsmaskingcategory.MaskingCategory, value string) string {
	// an empty value has nothing to hide, and tokenising it would make a zero Browser non-zero —
	// which is how IsBrowser tells a browser sample from an ordinary one
	if strings.TrimSpace(value) == "" {
		return value
	}

	masked, ok := tables.Values[MaskValue{MaskingCategory: maskingCategory, Value: value}]
	assert.True(ok)
	if !ok {
		return MaskedPlaceholder
	}

	return masked
}

func (tables MaskingTables) maskPointer(maskingCategory enumsmaskingcategory.MaskingCategory, value *string) *string {
	if value == nil {
		return nil
	}

	masked := tables.mask(maskingCategory, *value)

	return &masked
}

func (tables MaskingTables) maskCategory(category enumscategories.Category) enumscategories.Category {
	if category == enumscategories.CategoryUnknown {
		return category
	}

	masked, ok := tables.Categories[category]
	assert.True(ok)
	if !ok {
		return enumscategories.CategoryUnknown
	}

	return masked
}

func (tables MaskingTables) maskCategoryPointer(
	category *enumscategories.Category,
) *enumscategories.Category {
	if category == nil {
		return nil
	}

	masked := tables.maskCategory(*category)

	return &masked
}

// MaskInputs sits beside Mask so the two cannot drift: a field added to one and not the other masks
// to MaskedPlaceholder. Categories are absent — the permutation covers the taxonomy, not a process.
func (foregroundProcess ForegroundProcess) MaskInputs() []MaskValue {
	appMetadata := foregroundProcess.Enrichments.Appmetadata
	browser := foregroundProcess.Enrichments.Browser

	values := make([]MaskValue, 0)
	appendValue := func(maskingCategory enumsmaskingcategory.MaskingCategory, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}

		values = append(values, MaskValue{MaskingCategory: maskingCategory, Value: value})
	}

	appendValue(enumsmaskingcategory.AppName, utils.Coalesce(foregroundProcess.AppName, ""))
	appendValue(enumsmaskingcategory.AppIdentifier, utils.Coalesce(foregroundProcess.AppIdentifier, ""))
	appendValue(enumsmaskingcategory.AppFriendlyName, appMetadata.FriendlyName)
	appendValue(enumsmaskingcategory.BrowserVendor, browser.Vendor)
	appendValue(enumsmaskingcategory.BrowserAppIdentifier, utils.Coalesce(browser.AppIdentifier, ""))
	appendValue(enumsmaskingcategory.BrowserDomain, browser.Domain)

	return values
}

// Mask returns a counterpart safe to hand somewhere the real process must not go. Presence survives
// throughout, so a masked idle sample still satisfies IsIdle.
//
// Egress only: it zeroes PID and WindowTitle, the two fields IsEqual compares, so every masked
// non-idle sample compares equal. Never feed one to CollectTimeline.
func (foregroundProcess ForegroundProcess) Mask(tables MaskingTables) ForegroundProcess {
	appMetadata := foregroundProcess.Enrichments.Appmetadata
	browser := foregroundProcess.Enrichments.Browser
	location := foregroundProcess.Enrichments.Location

	return ForegroundProcess{
		AppName:       tables.maskPointer(enumsmaskingcategory.AppName, foregroundProcess.AppName),
		AppIdentifier: tables.maskPointer(enumsmaskingcategory.AppIdentifier, foregroundProcess.AppIdentifier),
		AppPath:       utils.Blank(foregroundProcess.AppPath),
		PID:           utils.Blank(foregroundProcess.PID),
		WindowTitle:   utils.Blank(foregroundProcess.WindowTitle),
		TitleSource:   foregroundProcess.TitleSource,
		Timestamp:     foregroundProcess.Timestamp,
		Idle:          foregroundProcess.Idle,
		Killed:        foregroundProcess.Killed,
		Enrichments: Enrichments{
			Appmetadata: AppMetadata{
				FriendlyName: tables.mask(enumsmaskingcategory.AppFriendlyName, appMetadata.FriendlyName),
				Description:  appMetadata.Description,
				Category:     tables.maskCategory(appMetadata.Category),
				IconPath:     "",
				Source:       appMetadata.Source,
			},
			Browser: Browser{
				Vendor:        tables.mask(enumsmaskingcategory.BrowserVendor, browser.Vendor),
				Category:      tables.maskCategoryPointer(browser.Category),
				AppIdentifier: tables.maskPointer(enumsmaskingcategory.BrowserAppIdentifier, browser.AppIdentifier),
				Tab:           "",
				CdpURL:        "",
				Domain:        tables.mask(enumsmaskingcategory.BrowserDomain, browser.Domain),
			},
			Location: Location{
				Latitude:  utils.Blank(location.Latitude),
				Longitude: utils.Blank(location.Longitude),
				PublicIP:  utils.Blank(location.PublicIP),
			},
		},
	}
}
