package transitions

import (
	"time"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

const subscriberBufferSize = 64

type Event struct {
	ID                    int64
	ApplicationName       string
	ApplicationIdentifier string
	ApplicationPath       string
	PID                   int32
	Reason                string
	StartedAt             time.Time
	EndedAt               time.Time
	Browser               bool
	Tab                   string
	Idle                  bool
	CDPURL                string
}

type Filters struct {
	StartedAt *time.Time
	EndedAt   *time.Time
}

type GetTransitionEventsParams struct {
	Filters Filters
}

type SubscribeParams struct {
	Filters Filters
}

type PublishTransitionEventParams struct {
	ID int64
}

type RecordTransitionEventParams struct {
	ApplicationName       string
	ApplicationIdentifier string
	ApplicationPath       string
	PID                   int32
	Reason                string
	StartedAt             time.Time
	EndedAt               time.Time
	Browser               bool
	Tab                   *string
	Idle                  bool
	CDPURL                *string
	Latitude              *float64
	Longitude             *float64
	PublicIP              *string
}

type Subscription struct {
	Events <-chan Event
	Close  func()
}

type subscriber struct {
	filters Filters
	events  chan Event
}

func eventMatchesFilters(event Event, filters Filters) bool {
	if filters.StartedAt != nil && event.StartedAt.Before(*filters.StartedAt) {
		return false
	}
	if filters.EndedAt != nil && event.EndedAt.After(*filters.EndedAt) {
		return false
	}

	return true
}

func eventFromGetTransitionEventsRow(row readqueries.GetTransitionEventsRow) Event {
	// Reason is not stored; the caller derives it from adjacency (see reasonForTransitionRow).
	event := Event{
		ID:        row.TransitionEventID,
		StartedAt: row.StartedAt.UTC(),
		EndedAt:   row.EndedAt.UTC(),
		PID:       int32(utils.Coalesce(row.Pid, 0)),
	}
	if row.ApplicationName != nil {
		event.ApplicationName = *row.ApplicationName
	}
	if row.ApplicationIdentifier != nil {
		event.ApplicationIdentifier = *row.ApplicationIdentifier
	}
	if row.ApplicationPath != nil {
		event.ApplicationPath = *row.ApplicationPath
	}
	if row.Browser != nil {
		event.Browser = *row.Browser
	}
	if row.Tab != nil {
		event.Tab = *row.Tab
	}
	if row.Idle != nil {
		event.Idle = *row.Idle
	}
	if row.CdpUrl != nil {
		event.CDPURL = *row.CdpUrl
	}

	return event
}

func eventFromGetTransitionEventRow(row readqueries.GetTransitionEventRow) Event {
	// Single-row lookup has no adjacency context, so reason is derived coarsely from the idle flag.
	event := Event{
		ID:        row.TransitionEventID,
		Reason:    coarseReason(row.Idle),
		StartedAt: row.StartedAt.UTC(),
		EndedAt:   row.EndedAt.UTC(),
		PID:       int32(utils.Coalesce(row.Pid, 0)),
	}
	if row.ApplicationName != nil {
		event.ApplicationName = *row.ApplicationName
	}
	if row.ApplicationIdentifier != nil {
		event.ApplicationIdentifier = *row.ApplicationIdentifier
	}
	if row.ApplicationPath != nil {
		event.ApplicationPath = *row.ApplicationPath
	}
	if row.Browser != nil {
		event.Browser = *row.Browser
	}
	if row.Tab != nil {
		event.Tab = *row.Tab
	}
	if row.Idle != nil {
		event.Idle = *row.Idle
	}
	if row.CdpUrl != nil {
		event.CDPURL = *row.CdpUrl
	}

	return event
}

// eventFromGetOpenTimelineEventRow maps the open (current) timeline row to an Event. The entry has no
// end boundary yet, so EndedAt is left zero for the caller to fill (e.g. usage clips it to "now").
func eventFromGetOpenTimelineEventRow(row readqueries.GetOpenTimelineEventRow) Event {
	event := Event{
		ID:        row.TransitionEventID,
		Reason:    reasonActive,
		StartedAt: row.StartedAt.UTC(),
		PID:       int32(utils.Coalesce(row.Pid, 0)),
	}
	if row.ApplicationName != nil {
		event.ApplicationName = *row.ApplicationName
	}
	if row.ApplicationIdentifier != nil {
		event.ApplicationIdentifier = *row.ApplicationIdentifier
	}
	if row.ApplicationPath != nil {
		event.ApplicationPath = *row.ApplicationPath
	}
	if row.Browser != nil {
		event.Browser = *row.Browser
	}
	if row.Tab != nil {
		event.Tab = *row.Tab
	}
	if row.Idle != nil {
		event.Idle = *row.Idle
	}
	if row.CdpUrl != nil {
		event.CDPURL = *row.CdpUrl
	}

	return event
}

// Transition reasons are derived at read time (they are no longer stored on the event store).
const (
	reasonStart          = "start"
	reasonIdle           = "idle"
	reasonReturnFromIdle = "return_from_idle"
	reasonTabChange      = "tab_change"
	reasonFocusChange    = "focus_change"
	reasonActive         = "active"
)

// coarseReason derives a reason without adjacency context (single-row lookups).
func coarseReason(idle *bool) string {
	if idle != nil && *idle {
		return reasonIdle
	}
	return reasonFocusChange
}

// reasonForTransitionRow derives the transition reason for row i of an ordered result set by
// comparing it with the previous row: idle -> idle, coming out of idle -> return_from_idle, same
// application as the previous entry -> tab_change, otherwise focus_change.
func reasonForTransitionRow(rows []readqueries.GetTransitionEventsRow, i int) string {
	cur := rows[i]
	if cur.Idle != nil && *cur.Idle {
		return reasonIdle
	}
	if i == 0 {
		return reasonFocusChange
	}
	prev := rows[i-1]
	if prev.Idle != nil && *prev.Idle {
		return reasonReturnFromIdle
	}
	if cur.ApplicationID != nil && prev.ApplicationID != nil && *cur.ApplicationID == *prev.ApplicationID {
		return reasonTabChange
	}
	return reasonFocusChange
}
