package appmetadata

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"
)

func TestLoadDesktopEntry(t *testing.T) {
	cfg, err := ini.LoadSources(
		ini.LoadOptions{IgnoreInlineComment: true, SkipUnrecognizableLines: true},
		filepath.Join("testdata", "org.videolan.VLC.desktop"),
	)
	require.NoError(t, err, "load")
	var entry desktopEntry
	require.NoError(t, cfg.Section("Desktop Entry").MapTo(&entry), "map")

	assert.Equal(t, "VLC media player", entry.Name) // unlocalized Name, not Name[de]
	assert.Equal(t, "Read, capture, broadcast your multimedia streams", entry.Comment)
	assert.Equal(t, "AudioVideo;Player;Recorder;", entry.Categories)
	assert.Equal(t, "vlc", entry.Icon)
	assert.Equal(t, "vlc", entry.StartupWMClass)

	category, _ := ParseFreedesktopCategories(entry.Categories)
	assert.Equal(t, CategoryMedia, category)
}

func TestDesktopMatches(t *testing.T) {
	entry := desktopEntry{StartupWMClass: "vlc", Exec: "/usr/bin/vlc %U"}
	cases := []struct {
		name    string
		base    string
		idLower string
		exeBase string
		want    bool
	}{
		{"wmclass", "org.videolan.VLC", "vlc", "", true},
		{"basename", "vlc", "vlc", "", true},
		{"last-dot-segment", "org.foo.code", "code", "", true},
		{"exec-base", "org.videolan.VLC", "", "vlc", true},
		{"no-match", "org.videolan.VLC", "firefox", "firefox", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, desktopMatches(entry, tc.base, tc.idLower, tc.exeBase))
		})
	}
}

func TestExecBase(t *testing.T) {
	cases := map[string]string{
		"/usr/bin/vlc --started-from-file %U": "vlc",
		"code %F":                             "code",
		`"/opt/app/My App" %u`:                "My App",
		"":                                    "",
	}
	for in, want := range cases {
		assert.Equalf(t, want, execBase(in), "execBase(%q)", in)
	}
}

func TestCategoryFromFreedesktop(t *testing.T) {
	cases := map[string]Category{
		"AudioVideo;Player;":        CategoryMedia,
		"Development;IDE;":          CategoryDevelopment,
		"Network;WebBrowser;":       CategoryWebBrowsing,
		"Network;InstantMessaging;": CategoryCommunication, // refining token wins over Network
		"Office;":                   CategoryProductivity,
		"Game;":                     CategoryGames,
		"Settings;System;":          CategorySystem,
		"NonsenseCategory;":         CategoryUnknown,
	}
	for cats, want := range cases {
		got, _ := ParseFreedesktopCategories(cats)
		assert.Equalf(t, want, got, "ParseFreedesktopCategories(%q)", cats)
	}
}

func TestCategoryFromApple(t *testing.T) {
	cases := map[string]Category{
		"public.app-category.developer-tools":   CategoryDevelopment,
		"public.app-category.productivity":      CategoryProductivity,
		"public.app-category.games":             CategoryGames,
		"public.app-category.action-games":      CategoryGames,
		"public.app-category.social-networking": CategorySocial,
		"public.app-category.unknown-thing":     CategoryUnknown,
		"":                                      CategoryUnknown,
	}
	for uti, want := range cases {
		got, err := ParseAppleCategory(uti)
		assert.Equalf(t, want, got, "ParseAppleCategory(%q)", uti)
		if want == CategoryUnknown {
			assert.Errorf(t, err, "ParseAppleCategory(%q) should error", uti)
		} else {
			assert.NoErrorf(t, err, "ParseAppleCategory(%q)", uti)
		}
	}
}
