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
	return []string{"unknown", "event", "day_summary", "app_day_summary", "time_block_summary"}[documentType]
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
