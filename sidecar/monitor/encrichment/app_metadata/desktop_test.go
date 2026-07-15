package app_metadata

import (
	"path/filepath"
	"testing"
)

func TestParseDesktopFile(t *testing.T) {
	entry, err := parseDesktopFile(filepath.Join("testdata", "org.videolan.VLC.desktop"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if entry.Name != "VLC media player" { // unlocalized Name, not Name[de]
		t.Errorf("Name = %q", entry.Name)
	}
	if entry.Comment != "Read, capture, broadcast your multimedia streams" {
		t.Errorf("Comment = %q", entry.Comment)
	}
	if entry.Categories != "AudioVideo;Player;Recorder;" {
		t.Errorf("Categories = %q", entry.Categories)
	}
	if entry.Icon != "vlc" {
		t.Errorf("Icon = %q", entry.Icon)
	}
	if entry.StartupWMClass != "vlc" {
		t.Errorf("StartupWMClass = %q", entry.StartupWMClass)
	}

	if code, _ := CategoryFromFreedesktop(entry.Categories); code != CategoryMedia {
		t.Errorf("category = %q, want %q", code, CategoryMedia)
	}
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
			if got := desktopMatches(entry, tc.base, tc.idLower, tc.exeBase); got != tc.want {
				t.Errorf("desktopMatches = %v, want %v", got, tc.want)
			}
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
		if got := execBase(in); got != want {
			t.Errorf("execBase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCategoryFromFreedesktop(t *testing.T) {
	cases := map[string]string{
		"AudioVideo;Player;":            CategoryMedia,
		"Development;IDE;":              CategoryDevelopment,
		"Network;WebBrowser;":          CategoryWebBrowsing,
		"Network;InstantMessaging;":    CategoryCommunication, // refining token wins over Network
		"Office;":                      CategoryProductivity,
		"Game;":                        CategoryGames,
		"Settings;System;":             CategorySystem,
		"NonsenseCategory;":            "",
	}
	for cats, want := range cases {
		if got, _ := CategoryFromFreedesktop(cats); got != want {
			t.Errorf("CategoryFromFreedesktop(%q) = %q, want %q", cats, got, want)
		}
	}
}

func TestCategoryFromApple(t *testing.T) {
	cases := map[string]string{
		"public.app-category.developer-tools": CategoryDevelopment,
		"public.app-category.productivity":    CategoryProductivity,
		"public.app-category.games":           CategoryGames,
		"public.app-category.action-games":    CategoryGames,
		"public.app-category.social-networking": CategorySocial,
		"public.app-category.unknown-thing":   "",
		"":                                    "",
	}
	for uti, want := range cases {
		if got, _ := CategoryFromApple(uti); got != want {
			t.Errorf("CategoryFromApple(%q) = %q, want %q", uti, got, want)
		}
	}
}
