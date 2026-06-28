package semantic

import "testing"

func TestDiversifySearchResultsPrefersSummariesForBroadQuestions(t *testing.T) {
	results := diversifySearchResults("What did I spend time on today?", []SearchResult{
		{DocumentKey: "event:1", DocumentType: DocumentTypeEvent},
		{DocumentKey: "app_day:2026-06-28:java", DocumentType: DocumentTypeAppDay},
		{DocumentKey: "day:2026-06-28", DocumentType: DocumentTypeDaySummary},
		{DocumentKey: "time_block:2026-06-28:afternoon", DocumentType: DocumentTypeTimeBlock},
	}, 3)

	if len(results) != 3 {
		t.Fatalf("expected three results, got %d", len(results))
	}
	if results[0].DocumentType != DocumentTypeDaySummary {
		t.Fatalf("expected day summary first, got %+v", results[0])
	}
	if results[1].DocumentType != DocumentTypeTimeBlock {
		t.Fatalf("expected time block second, got %+v", results[1])
	}
	if results[2].DocumentType != DocumentTypeAppDay {
		t.Fatalf("expected app-day summary third, got %+v", results[2])
	}
}

func TestDiversifySearchResultsPrefersEventsForSpecificQuestions(t *testing.T) {
	results := diversifySearchResults("Did I use java?", []SearchResult{
		{DocumentKey: "app_day:2026-06-28:java", DocumentType: DocumentTypeAppDay},
		{DocumentKey: "day:2026-06-28", DocumentType: DocumentTypeDaySummary},
		{DocumentKey: "event:1", DocumentType: DocumentTypeEvent},
	}, 2)

	if len(results) != 2 {
		t.Fatalf("expected two results, got %d", len(results))
	}
	if results[0].DocumentType != DocumentTypeEvent {
		t.Fatalf("expected event first, got %+v", results[0])
	}
	if results[1].DocumentType != DocumentTypeAppDay {
		t.Fatalf("expected app-day summary second, got %+v", results[1])
	}
}
