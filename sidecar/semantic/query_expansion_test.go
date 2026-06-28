package semantic

import (
	"strings"
	"testing"
	"time"
)

func TestExpandSemanticQueryAddsLocalDateAndTimeContext(t *testing.T) {
	loc := time.FixedZone("NZST", 12*60*60)
	now := time.Date(2026, 6, 28, 15, 30, 0, 0, loc)

	expanded := ExpandSemanticQuery("What did I spend time on this afternoon today?", now, loc)

	assertContains(t, expanded, "Original question: What did I spend time on this afternoon today?")
	assertContains(t, expanded, "today means Sunday, June 28, 2026 from 12:00 AM to 12:00 AM local time.")
	assertContains(t, expanded, "this afternoon means Sunday, June 28, 2026 from 12:00 PM to 6:00 PM local time.")
}

func TestExpandSemanticQueryLeavesPlainQuestionUntouched(t *testing.T) {
	question := "Did I use an app called java?"

	expanded := ExpandSemanticQuery(question, time.Now(), time.Local)

	if strings.Contains(expanded, "Local semantic query context") {
		t.Fatalf("unexpected query expansion: %s", expanded)
	}
	if expanded != question {
		t.Fatalf("expected original question %q, got %q", question, expanded)
	}
}
