package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/negrel/assert"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type TitleSource int

const (
	TitleSourceUnknown TitleSource = iota
	TitleSourceAX
	TitleSourceOsascript
	TitleSourceWindowAPI
	TitleSourceNone
)

func (titlteSource TitleSource) String() string {
	return []string{"unknown", "ax", "osascript", "window_api", "none"}[titlteSource]
}

func ParseTitleSource(code string) (TitleSource, error) {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "ax":
		return TitleSourceAX, nil
	case "osascript":
		return TitleSourceOsascript, nil
	case "window_api":
		return TitleSourceWindowAPI, nil
	case "none":
		return TitleSourceNone, nil
	}

	return TitleSourceUnknown, fmt.Errorf("unrecognized title source: %s", code)
}

type ForegroundProcess struct {
	AppName       *string
	AppIdentifier *string
	AppPath       *string
	PID           *int64
	WindowTitle   *string
	TitleSource   *TitleSource
	Timestamp     time.Time
	Idle          bool
	Killed        bool
	Enrichments   Enrichments
}

type Enrichments struct {
	Appmetadata AppMetadata
	Browser     Browser
	Location    Location
}

type AppMetadata struct {
	FriendlyName string
	Description  string
	Category     enumscategories.Category
	IconPath     string
	Source       string
}

type Browser struct {
	Vendor        string
	Category      *enumscategories.Category
	AppIdentifier *string
	Tab           string
	CdpURL        string
	Domain        string
}

type Location struct {
	Latitude  *float64
	Longitude *float64
	PublicIP  *string
}

type Usage struct {
	ForegroundProcess ForegroundProcess
	Killed            bool
}

// UsageSeq is one timeline entry: the observation that opened it and the one that closed it. End is
// nil while the entry is still open. ID is the timeline entry id — it is the identity downstream
// consumers report and deduplicate on, and is left zero by the purely in-memory SeqTimeline.
type UsageSeq struct {
	ID     int64
	Start  *ForegroundProcess
	End    *ForegroundProcess
	Killed bool
}

func (foregroundProcess ForegroundProcess) IsIdle() bool {
	if !foregroundProcess.Idle {
		return false
	}

	assert.Nil(foregroundProcess.AppName)
	assert.Nil(foregroundProcess.AppIdentifier)
	assert.Nil(foregroundProcess.AppPath)
	assert.Nil(foregroundProcess.PID)
	assert.Nil(foregroundProcess.WindowTitle)
	assert.Nil(foregroundProcess.TitleSource)

	assert.NotZero(foregroundProcess.Timestamp)

	return foregroundProcess.Idle
}

func (foregroundProcess ForegroundProcess) IsBrowser() bool {
	return !utils.IsZero(foregroundProcess.Enrichments.Browser)
}

func (foregroundProcess ForegroundProcess) IsEqual(x ForegroundProcess) bool {
	if foregroundProcess.Idle != x.Idle {
		return false
	}

	if foregroundProcess.Killed != x.Killed {
		return false
	}

	if foregroundProcess.PID != nil && x.PID != nil {
		if *foregroundProcess.PID != *x.PID {
			return false
		}

		if utils.Coalesce(foregroundProcess.WindowTitle, "") != utils.Coalesce(x.WindowTitle, "") { // browsers
			return false
		}
	}

	return true
}

func (usage Usage) IsKilled() bool {
	return usage.Killed
}

// Title is what an observation is reported as. A browser prefers the tab, because the tab is what the
// time was actually spent on; everything else prefers the application's name.
func (foregroundProcess ForegroundProcess) Title() string {
	switch {
	case foregroundProcess.Idle:
		return "Idle"
	case foregroundProcess.IsBrowser():
		return utils.Coalesce(utils.Or(func(x string) bool { return !utils.IsEmptyString(x) },
			strings.TrimSpace(foregroundProcess.Enrichments.Browser.Tab),
			foregroundProcess.ApplicationName(),
			"Browser",
		), "")
	default:
		return utils.Coalesce(utils.Or(func(x string) bool { return !utils.IsEmptyString(x) },
			foregroundProcess.ApplicationName(), "Application"), "")
	}
}

// SourceName is what was in focus, as opposed to what it was showing: a browser reports the browser
// rather than the tab.
func (foregroundProcess ForegroundProcess) SourceName() string {
	switch {
	case foregroundProcess.Idle:
		return "Idle"
	case foregroundProcess.IsBrowser():
		return utils.Coalesce(utils.Or(func(x string) bool { return !utils.IsEmptyString(x) },
			foregroundProcess.ApplicationName(), "Browser"), "")
	default:
		return utils.Coalesce(utils.Or(func(x string) bool { return !utils.IsEmptyString(x) },
			foregroundProcess.ApplicationName(), "Application"), "")
	}
}

func (foregroundProcess ForegroundProcess) ApplicationName() string {
	return strings.TrimSpace(utils.Coalesce(foregroundProcess.AppName, ""))
}

// ApplicationKey discriminates "same application, different window" from a real focus change.
// applications.identifier is the natural key of an application, with the name as a fallback for the
// rows where it was never captured.
func (foregroundProcess ForegroundProcess) ApplicationKey() string {
	if identifier := strings.TrimSpace(utils.Coalesce(foregroundProcess.AppIdentifier, "")); identifier != "" {
		return identifier
	}

	return foregroundProcess.ApplicationName()
}

// Span is the stretch an entry covers. It is zero-ended while the entry is still open.
func (seq UsageSeq) Span() utils.TimeSpan {
	span := utils.TimeSpan{}
	if seq.Start != nil {
		span[0] = seq.Start.Timestamp
	}
	if seq.End != nil {
		span[1] = seq.End.Timestamp
	}

	return span
}
