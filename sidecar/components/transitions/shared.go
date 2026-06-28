package transitions

import (
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

const subscriberBufferSize = 64

type Event struct {
	ID                    int64
	ApplicationName       string
	ApplicationIdentifier string
	ApplicationPath       string
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

func eventFromGetTransitionEventsRow(row queries.GetTransitionEventsRow) Event {
	event := Event{
		ID:        row.TransitionEventID,
		Reason:    row.Reason,
		StartedAt: row.StartedAt,
		EndedAt:   row.EndedAt,
	}
	if row.ApplicationName.Valid {
		event.ApplicationName = row.ApplicationName.String
	}
	if row.ApplicationPlatformIdentifier.Valid {
		event.ApplicationIdentifier = row.ApplicationPlatformIdentifier.String
	}
	if row.ApplicationPath.Valid {
		event.ApplicationPath = row.ApplicationPath.String
	}
	if row.Browser.Valid {
		event.Browser = row.Browser.Bool
	}
	if row.Tab != nil {
		event.Tab = *row.Tab
	}
	if row.Idle.Valid {
		event.Idle = row.Idle.Bool
	}
	if row.CdpUrl != nil {
		event.CDPURL = *row.CdpUrl
	}

	return event
}

func eventFromGetTransitionEventRow(row queries.GetTransitionEventRow) Event {
	event := Event{
		ID:        row.TransitionEventID,
		Reason:    row.Reason,
		StartedAt: row.StartedAt,
		EndedAt:   row.EndedAt,
	}
	if row.ApplicationName.Valid {
		event.ApplicationName = row.ApplicationName.String
	}
	if row.ApplicationPlatformIdentifier.Valid {
		event.ApplicationIdentifier = row.ApplicationPlatformIdentifier.String
	}
	if row.ApplicationPath.Valid {
		event.ApplicationPath = row.ApplicationPath.String
	}
	if row.Browser.Valid {
		event.Browser = row.Browser.Bool
	}
	if row.Tab != nil {
		event.Tab = *row.Tab
	}
	if row.Idle.Valid {
		event.Idle = row.Idle.Bool
	}
	if row.CdpUrl != nil {
		event.CDPURL = *row.CdpUrl
	}

	return event
}
