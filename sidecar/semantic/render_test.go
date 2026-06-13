package semantic

import (
	"database/sql"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

func TestRenderTransitionEventDocument(t *testing.T) {
	tab := "GitHub"
	url := "https://github.com/"
	content := RenderTransitionEventDocument(queries.GetTransitionEventDocumentSourceRow{
		TransitionEventID: 1,
		ApplicationName:   sql.NullString{String: "Google Chrome", Valid: true},
		Reason:            "tab_change",
		StartedAt:         time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
		EndedAt:           time.Date(2026, 6, 13, 10, 5, 0, 0, time.UTC),
		Browser:           sql.NullBool{Bool: true, Valid: true},
		Tab:               &tab,
		Idle:              sql.NullBool{Bool: false, Valid: true},
		CdpUrl:            &url,
	})

	expected := "Reason: tab_change\n" +
		"Application: Google Chrome\n" +
		"Started at: 2026-06-13T10:00:00Z\n" +
		"Ended at: 2026-06-13T10:05:00Z\n" +
		"Browser: true\n" +
		"Tab: GitHub\n" +
		"URL: https://github.com/\n" +
		"Idle: false"
	if content != expected {
		t.Fatalf("unexpected content:\n%s", content)
	}
}
