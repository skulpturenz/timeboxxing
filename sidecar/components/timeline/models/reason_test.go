package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ptr[T any](v T) *T {
	return &v
}

func observation(name string, identifier string) ForegroundProcess {
	return ForegroundProcess{
		AppName:       ptr(name),
		AppIdentifier: ptr(identifier),
		PID:           ptr(int64(1)),
		Timestamp:     time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC),
	}
}

func entry(start ForegroundProcess) UsageSeq {
	end := start
	end.Timestamp = start.Timestamp.Add(time.Minute)

	return UsageSeq{Start: &start, End: &end}
}

// A reason is derived by comparing an entry with its predecessor, never read back from a column.
func TestReasonFor_ClassifiesTransitions(t *testing.T) {
	ghostty := entry(observation("Ghostty", "com.ghostty"))
	slack := entry(observation("Slack", "com.slack"))
	otherGhosttyWindow := entry(observation("Ghostty", "com.ghostty"))
	idle := entry(ForegroundProcess{Idle: true, Timestamp: ghostty.Start.Timestamp})

	tests := []struct {
		name     string
		seq      UsageSeq
		previous *UsageSeq
		expected Reason
	}{
		{name: "the first entry has nothing to compare against", seq: ghostty, expected: ReasonFocusChange},
		{name: "a different application is a focus change", seq: slack, previous: &ghostty, expected: ReasonFocusChange},
		{name: "the same application is a window or tab change", seq: otherGhosttyWindow, previous: &ghostty, expected: ReasonTabChange},
		{name: "an idle entry is idle whatever preceded it", seq: idle, previous: &ghostty, expected: ReasonIdle},
		{name: "the entry after idle is a return from it", seq: ghostty, previous: &idle, expected: ReasonReturnFromIdle},
		{name: "an entry without an opening observation is unknown", seq: UsageSeq{}, previous: &ghostty, expected: ReasonUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, ReasonFor(test.seq, test.previous))
		})
	}
}

func TestReason_RoundTripsThroughItsCode(t *testing.T) {
	for _, reason := range []Reason{ReasonIdle, ReasonReturnFromIdle, ReasonTabChange, ReasonFocusChange} {
		parsed, err := ParseReason(reason.String())
		assert.NoError(t, err)
		assert.Equal(t, reason, parsed)
	}

	_, err := ParseReason("active")
	assert.Error(t, err, "the running session is no longer part of the vocabulary")
}

// Title prefers what the time was spent on; SourceName prefers what was in focus.
func TestForegroundProcess_TitleAndSourceName(t *testing.T) {
	browser := observation("Google Chrome", "com.google.Chrome")
	browser.Enrichments.Browser = Browser{Vendor: "Google Chrome", Tab: "Pull requests"}

	assert.Equal(t, "Pull requests", browser.Title())
	assert.Equal(t, "Google Chrome", browser.SourceName())

	untitledBrowser := observation("Google Chrome", "com.google.Chrome")
	untitledBrowser.Enrichments.Browser = Browser{Vendor: "Google Chrome"}
	assert.Equal(t, "Google Chrome", untitledBrowser.Title(), "a browser without a tab falls back to its name")

	anonymous := ForegroundProcess{Enrichments: Enrichments{Browser: Browser{Vendor: "Chrome"}}}
	assert.Equal(t, "Browser", anonymous.Title())

	application := observation("Ghostty", "com.ghostty")
	assert.Equal(t, "Ghostty", application.Title())
	assert.Equal(t, "Ghostty", application.SourceName())

	assert.Equal(t, "Application", ForegroundProcess{}.Title())
	assert.Equal(t, "Idle", ForegroundProcess{Idle: true}.Title())
}

// The identifier is an application's natural key, with the name as a fallback for the rows where it
// was never captured.
func TestForegroundProcess_ApplicationKey(t *testing.T) {
	assert.Equal(t, "com.ghostty", observation("Ghostty", "com.ghostty").ApplicationKey())
	assert.Equal(t, "Ghostty", ForegroundProcess{AppName: ptr("Ghostty")}.ApplicationKey())
	assert.Empty(t, ForegroundProcess{}.ApplicationKey())
}
