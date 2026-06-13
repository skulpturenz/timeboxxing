package browser

import "testing"

func TestIsBrowser(t *testing.T) {
	tests := []struct {
		appName string
		want    BrowserKind
	}{
		{"Google Chrome", BrowserChrome},
		{"google chrome", BrowserChrome},
		{"Chromium", BrowserChrome},
		{"Firefox", BrowserFirefox},
		{"Firefox Developer Edition", BrowserFirefox},
		{"Safari", BrowserSafari},
		{"Microsoft Edge", BrowserEdge},
		{"msedge", BrowserEdge},
		{"chrome", BrowserChrome},
		{"Visual Studio Code", BrowserNone},
		{"Terminal", BrowserNone},
		{"", BrowserNone},
		{"firefox", BrowserFirefox},
	}
	for _, tc := range tests {
		got := IsBrowser(tc.appName)
		if got != tc.want {
			t.Errorf("IsBrowser(%q) = %v, want %v", tc.appName, got, tc.want)
		}
	}
}

func TestParseTabTitle(t *testing.T) {
	tests := []struct {
		appName  string
		rawTitle string
		wantTab  string
		wantOK   bool
	}{
		// Chrome
		{"Google Chrome", "GitHub - Pull Request #42 - Google Chrome", "GitHub - Pull Request #42", true},
		{"Google Chrome", "Gmail - Google Chrome", "Gmail", true},
		{"Google Chrome", "New Tab - Google Chrome", "New Tab", true},
		{"Google Chrome", "Google Chrome", "New Tab", true}, // bare browser name → New Tab
		{"Google Chrome", "", "", false},

		// Firefox
		{"Firefox", "MDN Web Docs \u2014 Mozilla Firefox", "MDN Web Docs", true},
		{"Firefox", "Stack Overflow - Mozilla Firefox", "Stack Overflow", true},

		// Safari (no suffix appended)
		{"Safari", "Apple - macOS", "Apple - macOS", true},
		{"Safari", "GitHub", "GitHub", true},
		{"Safari", "", "", false},

		// Edge
		{"Microsoft Edge", "Bing - Microsoft Edge", "Bing", true},

		// Not a browser
		{"Terminal", "bash -- 80x24", "", false},
		{"Visual Studio Code", "main.go - project", "", false},

		// Chrome with em-dash variant
		{"Google Chrome", "GitHub \u2014 Google Chrome", "GitHub", true},
	}

	for _, tc := range tests {
		tab, ok := ParseTabTitle(tc.appName, tc.rawTitle)
		if ok != tc.wantOK {
			t.Errorf("ParseTabTitle(%q, %q): ok=%v, want %v", tc.appName, tc.rawTitle, ok, tc.wantOK)
			continue
		}
		if ok && tab.TabTitle != tc.wantTab {
			t.Errorf("ParseTabTitle(%q, %q): TabTitle=%q, want %q", tc.appName, tc.rawTitle, tab.TabTitle, tc.wantTab)
		}
	}
}
