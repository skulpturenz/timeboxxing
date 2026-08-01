package models

import (
	"fmt"
	"strings"
)

// Reason is why one stretch of activity gave way to the next. It is derived by comparing an entry
// with the one before it, never read back from a column.
type Reason int

const (
	ReasonUnknown Reason = iota
	ReasonIdle
	ReasonReturnFromIdle
	ReasonTabChange
	ReasonFocusChange
)

func (reason Reason) String() string {
	return []string{"unknown", "idle", "return_from_idle", "tab_change", "focus_change"}[reason]
}

func ParseReason(code string) (Reason, error) {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "idle":
		return ReasonIdle, nil
	case "return_from_idle":
		return ReasonReturnFromIdle, nil
	case "tab_change":
		return ReasonTabChange, nil
	case "focus_change":
		return ReasonFocusChange, nil
	}

	return ReasonUnknown, fmt.Errorf("unrecognized reason: %s", code)
}

// ReasonFor classifies an entry by comparing it with its predecessor. previous is the entry
// immediately before this one and is nil for the first entry in a stream, which has nothing to
// compare against and so reads as an ordinary focus change.
func ReasonFor(seq UsageSeq, previous *UsageSeq) Reason {
	if seq.Start == nil {
		return ReasonUnknown
	}

	if seq.Start.Idle {
		return ReasonIdle
	}

	if previous == nil || previous.Start == nil {
		return ReasonFocusChange
	}

	if previous.Start.Idle {
		return ReasonReturnFromIdle
	}

	if key := seq.Start.ApplicationKey(); key != "" && key == previous.Start.ApplicationKey() {
		return ReasonTabChange
	}

	return ReasonFocusChange
}
