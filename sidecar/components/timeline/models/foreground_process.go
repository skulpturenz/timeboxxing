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

type UsageSeq struct {
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
