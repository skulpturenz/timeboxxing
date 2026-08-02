package models

import (
	"fmt"
	"testing"
	"time"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	enumsmaskingcategory "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_masking_category"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func maskable() ForegroundProcess {
	titleSource := TitleSourceAX
	browserCategory := enumscategories.CategoryWebBrowsing
	latitude := -36.8485
	longitude := 174.7633

	return ForegroundProcess{
		AppName:       new("Ghostty"),
		AppIdentifier: new("com.mitchellh.ghostty"),
		AppPath:       new("/Users/ada/Applications/Ghostty.app"),
		PID:           new(int64(4321)),
		WindowTitle:   new("~/src/timeboxxing — zsh"),
		TitleSource:   &titleSource,
		Timestamp:     time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC),
		Idle:          false,
		Killed:        false,
		Enrichments: Enrichments{
			Appmetadata: AppMetadata{
				FriendlyName: "Ghostty",
				Description:  "A fast, native terminal emulator",
				Category:     enumscategories.CategoryDevelopment,
				IconPath:     "/Users/ada/Library/Caches/icons/ghostty.png",
				Source:       "homebrew",
			},
			Browser: Browser{
				Vendor:        "Google Chrome",
				Category:      &browserCategory,
				AppIdentifier: new("com.google.Chrome"),
				Tab:           "Pull requests · skulpturenz/timeboxxing",
				CdpURL:        "http://127.0.0.1:9222/devtools/page/ABC",
				Domain:        "github.com",
			},
			Location: Location{
				Latitude:  &latitude,
				Longitude: &longitude,
				PublicIP:  new("203.0.113.7"),
			},
		},
	}
}

func tablesFor(processes ...ForegroundProcess) MaskingTables {
	values := make(map[MaskValue]string)
	for _, process := range processes {
		for _, value := range process.MaskInputs() {
			values[value] = fmt.Sprintf("%s_%d", value.MaskingCategory, len(values))
		}
	}

	// assigned rather than written as a literal: a permutation is partial by nature, and a map
	// literal over an enum has to name every member
	categories := make(map[enumscategories.Category]enumscategories.Category)
	categories[enumscategories.CategoryDevelopment] = enumscategories.CategoryGames
	categories[enumscategories.CategoryWebBrowsing] = enumscategories.CategorySystem

	return MaskingTablesFrom(values, categories)
}

func TestMask_ReplacesIdentifyingTextWithItsToken(t *testing.T) {
	t.Parallel()

	process := maskable()
	tables := tablesFor(process)
	masked := process.Mask(tables)

	assert.Equal(t,
		tables.Values[MaskValue{MaskingCategory: enumsmaskingcategory.AppName, Value: "Ghostty"}],
		*masked.AppName, "the app name reads as its token")
	assert.Equal(t,
		tables.Values[MaskValue{MaskingCategory: enumsmaskingcategory.AppIdentifier, Value: "com.mitchellh.ghostty"}],
		*masked.AppIdentifier, "the bundle id reads as its token")
	assert.Equal(t,
		tables.Values[MaskValue{MaskingCategory: enumsmaskingcategory.AppFriendlyName, Value: "Ghostty"}],
		masked.Enrichments.Appmetadata.FriendlyName, "the friendly name reads as its token")
	assert.Equal(t,
		tables.Values[MaskValue{MaskingCategory: enumsmaskingcategory.BrowserVendor, Value: "Google Chrome"}],
		masked.Enrichments.Browser.Vendor, "the browser vendor reads as its token")
	assert.Equal(
		t,
		tables.Values[MaskValue{MaskingCategory: enumsmaskingcategory.BrowserAppIdentifier, Value: "com.google.Chrome"}],
		*masked.Enrichments.Browser.AppIdentifier,
		"the browser bundle id reads as its token",
	)
	assert.Equal(t,
		tables.Values[MaskValue{MaskingCategory: enumsmaskingcategory.BrowserDomain, Value: "github.com"}],
		masked.Enrichments.Browser.Domain, "the domain reads as its token")
}

func TestMask_IsStableAcrossCalls(t *testing.T) {
	t.Parallel()

	process := maskable()
	tables := tablesFor(process)
	first := process.Mask(tables)
	second := process.Mask(tables)

	assert.Equal(t, first, second, "masking twice agrees with itself")
}

func TestMask_BlanksWithoutErasingPresence(t *testing.T) {
	t.Parallel()

	masked := maskable().Mask(tablesFor(maskable()))

	require.NotNil(t, masked.AppPath, "a recorded path stays recorded")
	assert.Empty(t, *masked.AppPath, "the path itself is gone")
	require.NotNil(t, masked.PID, "a recorded pid stays recorded")
	assert.Zero(t, *masked.PID, "the pid itself is gone")
	require.NotNil(t, masked.WindowTitle, "a recorded window title stays recorded")
	assert.Empty(t, *masked.WindowTitle, "the title itself is gone")

	require.NotNil(t, masked.Enrichments.Location.Latitude, "a recorded latitude stays recorded")
	assert.Zero(t, *masked.Enrichments.Location.Latitude, "the latitude itself is gone")
	require.NotNil(t, masked.Enrichments.Location.Longitude, "a recorded longitude stays recorded")
	assert.Zero(t, *masked.Enrichments.Location.Longitude, "the longitude itself is gone")
	require.NotNil(t, masked.Enrichments.Location.PublicIP, "a recorded public ip stays recorded")
	assert.Empty(t, *masked.Enrichments.Location.PublicIP, "the ip itself is gone")

	assert.Empty(t, masked.Enrichments.Browser.Tab, "the tab is gone")
	assert.Empty(t, masked.Enrichments.Browser.CdpURL, "the cdp url is gone")
	assert.Empty(t, masked.Enrichments.Appmetadata.IconPath, "the icon path is gone")
}

func TestMask_LeavesNonIdentifyingFieldsAlone(t *testing.T) {
	t.Parallel()

	process := maskable()
	masked := process.Mask(tablesFor(process))

	assert.Equal(t, process.Timestamp, masked.Timestamp, "the instant is not identifying")
	assert.Equal(t, process.TitleSource, masked.TitleSource, "how the title was read is not identifying")
	assert.Equal(t, process.Killed, masked.Killed, "the killed flag is not identifying")
	assert.Equal(t, process.Enrichments.Appmetadata.Description, masked.Enrichments.Appmetadata.Description,
		"the catalog description is not user data")
	assert.Equal(t, process.Enrichments.Appmetadata.Source, masked.Enrichments.Appmetadata.Source,
		"the enrichment source is not user data")
}

func TestMask_KeepsAbsentFieldsAbsent(t *testing.T) {
	t.Parallel()

	process := processZero
	process.Timestamp = time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)
	masked := process.Mask(tablesFor(process))

	assert.Nil(t, masked.AppName, "an absent app name stays absent")
	assert.Nil(t, masked.AppIdentifier, "an absent bundle id stays absent")
	assert.Nil(t, masked.AppPath, "an absent path stays absent")
	assert.Nil(t, masked.PID, "an absent pid stays absent")
	assert.Nil(t, masked.WindowTitle, "an absent window title stays absent")
	assert.Nil(t, masked.Enrichments.Browser.Category, "an absent browser category stays absent")
	assert.Nil(t, masked.Enrichments.Browser.AppIdentifier, "an absent browser bundle id stays absent")
	assert.Nil(t, masked.Enrichments.Location.Latitude, "an absent latitude stays absent")
	assert.Nil(t, masked.Enrichments.Location.Longitude, "an absent longitude stays absent")
	assert.Nil(t, masked.Enrichments.Location.PublicIP, "an absent public ip stays absent")
}

func TestMask_IdleSampleStaysIdle(t *testing.T) {
	t.Parallel()

	process := processZero
	process.Idle = true
	process.Timestamp = time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)

	assert.True(t, process.Mask(tablesFor(process)).IsIdle(), "a masked idle stretch is still idle")
}

func TestMask_NonBrowserSampleStaysNonBrowser(t *testing.T) {
	t.Parallel()

	process := maskable()
	process.Enrichments.Browser = browserZero
	masked := process.Mask(tablesFor(process))

	require.True(t, maskable().Mask(tablesFor(maskable())).IsBrowser(), "a browser is still a browser")
	assert.False(t, masked.IsBrowser(), "an ordinary application does not become a browser")
}

func TestMask_PermutesCategories(t *testing.T) {
	t.Parallel()

	process := maskable()
	masked := process.Mask(tablesFor(process))

	assert.Equal(t, enumscategories.CategoryGames, masked.Enrichments.Appmetadata.Category,
		"the app category stands in for another")
	require.NotNil(t, masked.Enrichments.Browser.Category, "a recorded browser category stays recorded")
	assert.Equal(t, enumscategories.CategorySystem, *masked.Enrichments.Browser.Category,
		"the browser category stands in for another")
}

func TestMask_LeavesUnknownCategoryAlone(t *testing.T) {
	t.Parallel()

	process := maskable()
	process.Enrichments.Appmetadata.Category = enumscategories.CategoryUnknown
	masked := process.Mask(tablesFor(process))

	assert.Equal(t, enumscategories.CategoryUnknown, masked.Enrichments.Appmetadata.Category,
		"unknown maps to itself")
}

// A release build has the assertion compiled out and returns MaskedPlaceholder instead — that branch
// is therefore unreachable here, and untested.
func TestMask_FailsClosedOnAnUnissuedValue(t *testing.T) {
	t.Parallel()

	process := maskable()
	empty := MaskingTablesFrom(
		map[MaskValue]string{},
		map[enumscategories.Category]enumscategories.Category{},
	)

	require.Panics(t, func() { _ = process.Mask(empty) }, "an unissued value trips the assertion")
}

func TestMaskInputs_CoversEveryTokenisedField(t *testing.T) {
	t.Parallel()

	process := maskable()

	assert.ElementsMatch(t, []MaskValue{
		{MaskingCategory: enumsmaskingcategory.AppName, Value: "Ghostty"},
		{MaskingCategory: enumsmaskingcategory.AppIdentifier, Value: "com.mitchellh.ghostty"},
		{MaskingCategory: enumsmaskingcategory.AppFriendlyName, Value: "Ghostty"},
		{MaskingCategory: enumsmaskingcategory.BrowserVendor, Value: "Google Chrome"},
		{MaskingCategory: enumsmaskingcategory.BrowserAppIdentifier, Value: "com.google.Chrome"},
		{MaskingCategory: enumsmaskingcategory.BrowserDomain, Value: "github.com"},
	}, process.MaskInputs(), "every tokenised field declares itself")
}

func TestMaskInputs_SkipsEmptyValues(t *testing.T) {
	t.Parallel()

	process := processZero
	process.AppName = new("   ")

	assert.Empty(t, processZero.MaskInputs(), "a bare sample needs no tokens")
	assert.Empty(t, process.MaskInputs(), "a blank app name is nothing to hide")
}

func TestMaskingTables_UnmaskInvertsMask(t *testing.T) {
	t.Parallel()

	process := maskable()
	tables := tablesFor(process)
	masked := process.Mask(tables)

	name, ok := tables.Unmask(enumsmaskingcategory.AppName, *masked.AppName)
	require.True(t, ok, "a token this install issued resolves")
	assert.Equal(t, "Ghostty", name, "the token resolves to the real app name")

	domain, ok := tables.Unmask(enumsmaskingcategory.BrowserDomain, masked.Enrichments.Browser.Domain)
	require.True(t, ok, "a domain token resolves")
	assert.Equal(t, "github.com", domain, "the token resolves to the real domain")
}

func TestMaskingTables_UnmaskRejectsAnUnknownToken(t *testing.T) {
	t.Parallel()

	process := maskable()
	tables := tablesFor(process)
	masked := process.Mask(tables)

	_, ok := tables.Unmask(enumsmaskingcategory.BrowserDomain, *masked.AppName)
	assert.False(t, ok, "an app name token is not a domain token")

	_, ok = tables.Unmask(enumsmaskingcategory.AppName, "app_name_neverissued")
	assert.False(t, ok, "a token this install never issued does not resolve")
}

func TestParseMaskingCategory_InvertsString(t *testing.T) {
	t.Parallel()

	for _, maskingCategory := range []enumsmaskingcategory.MaskingCategory{
		enumsmaskingcategory.AppName,
		enumsmaskingcategory.AppIdentifier,
		enumsmaskingcategory.AppFriendlyName,
		enumsmaskingcategory.BrowserVendor,
		enumsmaskingcategory.BrowserAppIdentifier,
		enumsmaskingcategory.BrowserDomain,
	} {
		parsed, err := enumsmaskingcategory.Parse(maskingCategory.String())
		require.NoError(t, err, "every code the enum emits parses back")
		assert.Equal(t, maskingCategory, parsed, "the code round-trips to its member")
	}

	_, err := enumsmaskingcategory.Parse("unknown")
	assert.Error(t, err, "the sentinel is not a code a column may hold")
}
