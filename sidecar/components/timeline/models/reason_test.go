package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Zero values, declared rather than written as literals so the fixtures below stay readable.
var (
	browserZero Browser
	processZero ForegroundProcess
)

func observation(name string, identifier string) ForegroundProcess {
	process := processZero
	process.AppName = new(name)
	process.AppIdentifier = new(identifier)
	process.PID = new(int64(1))
	process.Timestamp = time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)

	return process
}

func entry(start ForegroundProcess) UsageSeq {
	end := start
	end.Timestamp = start.Timestamp.Add(time.Minute)

	return UsageSeq{ID: 0, Start: &start, End: &end, Killed: false}
}

// A reason is derived by comparing an entry with its predecessor, never read back from a column.
func TestReasonFor_ClassifiesTransitions(t *testing.T) {
	t.Parallel()

	ghostty := entry(observation("Ghostty", "com.ghostty"))
	slack := entry(observation("Slack", "com.slack"))
	otherGhosttyWindow := entry(observation("Ghostty", "com.ghostty"))
	idleProcess := processZero
	idleProcess.Idle = true
	idleProcess.Timestamp = ghostty.Start.Timestamp
	idle := entry(idleProcess)

	tests := []struct {
		name     string
		seq      UsageSeq
		previous *UsageSeq
		expected Reason
	}{
		{
			name:     "the first entry has nothing to compare against",
			seq:      ghostty,
			previous: nil,
			expected: ReasonFocusChange,
		},
		{
			name:     "a different application is a focus change",
			seq:      slack,
			previous: &ghostty,
			expected: ReasonFocusChange,
		},
		{
			name:     "the same application is a window or tab change",
			seq:      otherGhosttyWindow,
			previous: &ghostty,
			expected: ReasonTabChange,
		},
		{name: "an idle entry is idle whatever preceded it", seq: idle, previous: &ghostty, expected: ReasonIdle},
		{
			name:     "the entry after idle is a return from it",
			seq:      ghostty,
			previous: &idle,
			expected: ReasonReturnFromIdle,
		},
		{
			name:     "an entry without an opening observation is unknown",
			seq:      UsageSeq{ID: 0, Start: nil, End: nil, Killed: false},
			previous: &ghostty,
			expected: ReasonUnknown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expected, ReasonFor(test.seq, test.previous))
		})
	}
}

func TestReason_RoundTripsThroughItsCode(t *testing.T) {
	t.Parallel()

	for _, reason := range []Reason{ReasonIdle, ReasonReturnFromIdle, ReasonTabChange, ReasonFocusChange} {
		parsed, err := ParseReason(reason.String())
		require.NoError(t, err)
		assert.Equal(t, reason, parsed)
	}

	_, err := ParseReason("active")
	assert.Error(t, err, "the running session is no longer part of the vocabulary")
}

// Title prefers what the time was spent on; SourceName prefers what was in focus.
func TestForegroundProcess_TitleAndSourceName(t *testing.T) {
	t.Parallel()

	browser := observation("Google Chrome", "com.google.Chrome")
	browser.Enrichments.Browser = Browser{
		Vendor:        "Google Chrome",
		Category:      nil,
		AppIdentifier: nil,
		Tab:           "Pull requests",
		CdpURL:        "",
		Domain:        "",
	}

	assert.Equal(t, "Pull requests", browser.Title())
	assert.Equal(t, "Google Chrome", browser.SourceName())

	untitledBrowser := observation("Google Chrome", "com.google.Chrome")
	untitledBrowser.Enrichments.Browser = browserZero
	untitledBrowser.Enrichments.Browser.Vendor = "Google Chrome"
	assert.Equal(t, "Google Chrome", untitledBrowser.Title(), "a browser without a tab falls back to its name")

	anonymous := processZero
	anonymous.Enrichments.Browser.Vendor = "Chrome"
	assert.Equal(t, "Browser", anonymous.Title())

	application := observation("Ghostty", "com.ghostty")
	assert.Equal(t, "Ghostty", application.Title())
	assert.Equal(t, "Ghostty", application.SourceName())

	assert.Equal(t, "Application", processZero.Title())

	idleOnly := processZero
	idleOnly.Idle = true
	assert.Equal(t, "Idle", idleOnly.Title())
}

// The identifier is an application's natural key, with the name as a fallback for the rows where it
// was never captured.
func TestForegroundProcess_ApplicationKey(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "com.ghostty", observation("Ghostty", "com.ghostty").ApplicationKey())
	nameOnly := processZero
	nameOnly.AppName = new("Ghostty")
	assert.Equal(t, "Ghostty", nameOnly.ApplicationKey())
	assert.Empty(t, processZero.ApplicationKey())
}
