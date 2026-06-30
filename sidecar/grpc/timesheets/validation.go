package timesheets

import (
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	minutesPerDay           = 24 * 60
	maxEntryDurationMinutes = 12 * 60
	millisPerMinute         = int64(60_000)
)

func dayWindow(startedAt *timestamppb.Timestamp, endedAt *timestamppb.Timestamp) (time.Time, time.Time, bool) {
	if startedAt == nil || endedAt == nil {
		return time.Time{}, time.Time{}, false
	}
	start := startedAt.AsTime().UTC()
	end := endedAt.AsTime().UTC()
	return start, end, end.After(start)
}

func entryTitle(title string) string {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return "New time entry"
	}
	return trimmed
}

func projectIDOrEmpty(projectID string) string {
	return strings.TrimSpace(projectID)
}

func validEntryTime(startMinute int32, durationMinutes int32) bool {
	return startMinute >= 0 &&
		startMinute < minutesPerDay &&
		durationMinutes > 0 &&
		durationMinutes <= maxEntryDurationMinutes
}

func startedAtForEntry(dayStartedAt time.Time, startMinute int64) time.Time {
	return dayStartedAt.Add(time.Duration(startMinute) * time.Minute)
}

func durationMillis(durationMinutes int64) int64 {
	return durationMinutes * millisPerMinute
}
