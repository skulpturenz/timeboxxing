package enumssemanticdocumenttype

import (
	"fmt"
	"strings"
)

type SemanticDocumentType int

const (
	Unknown SemanticDocumentType = iota
	Event
	DaySummary
	AppDaySummary
	TimeBlockSummary
)

func (documentType SemanticDocumentType) String() string {
	switch documentType {
	case Event:
		return "event"
	case DaySummary:
		return "day_summary"
	case AppDaySummary:
		return "app_day_summary"
	case TimeBlockSummary:
		return "time_block_summary"
	default:
		return "unknown"
	}
}

func Parse(s string) (SemanticDocumentType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "event":
		return Event, nil
	case "day_summary":
		return DaySummary, nil
	case "app_day_summary":
		return AppDaySummary, nil
	case "time_block_summary":
		return TimeBlockSummary, nil
	}

	return Unknown, fmt.Errorf("unrecognized semantic document type: %s", s)
}
