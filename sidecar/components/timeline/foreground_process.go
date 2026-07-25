package timeline

import (
	"container/list"
	"fmt"
	"slices"
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
	PID           *int32
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
	Browser       string
	Category      *enumscategories.Category // TODO
	AppIdentifier *string                   // TODO
	Title         string
	Tab           string
	URL           string
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

func assertSorted(stream []ForegroundProcess) {
	assert.Condition(func() bool { // expect asc sort
		cloned := slices.Clone(stream)
		slices.SortFunc(cloned, func(x ForegroundProcess, y ForegroundProcess) int {
			return x.Timestamp.Compare(y.Timestamp) // asc
		})

		return slices.Equal(stream, cloned)
	})
}

func CollectTimeline(stream []ForegroundProcess) *list.List {
	list := list.New()

	assertSorted(stream)

	for _, v := range stream {
		prev := list.Back()
		if prev == nil || !prev.Value.(ForegroundProcess).IsEqual(v) {
			list.PushBack(v)
		}
	}

	return list
}

func SeqTimeline(timeline *list.List) []UsageSeq {
	seq := []UsageSeq{}

	for e := timeline.Front(); e != nil; e = e.Next() {
		prev := e.Prev()
		next := e.Next()

		var start *ForegroundProcess
		if prev != nil {
			v, ok := prev.Value.(ForegroundProcess)
			assert.True(ok)

			start = &v
		}

		var end *ForegroundProcess
		if next != nil {
			v, ok := next.Value.(ForegroundProcess)
			assert.True(ok)

			end = &v
		}

		current, ok := e.Value.(ForegroundProcess)
		assert.True(ok)

		item := UsageSeq{
			Start:  start,
			End:    end,
			Killed: current.Killed,
		}

		seq = append(seq, item)
	}

	return seq
}
