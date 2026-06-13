package semantic

import (
	"fmt"
	"strings"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

func RenderTransitionEventDocument(src queries.GetTransitionEventDocumentSourceRow) string {
	lines := []string{
		fmt.Sprintf("Reason: %s", src.Reason),
		fmt.Sprintf("Application: %s", nullStringValue(src.ApplicationName.Valid, src.ApplicationName.String, "None")),
		fmt.Sprintf("Started at: %s", formatTime(src.StartedAt)),
		fmt.Sprintf("Ended at: %s", formatTime(src.EndedAt)),
		fmt.Sprintf("Browser: %t", src.Browser.Valid && src.Browser.Bool),
		fmt.Sprintf("Tab: %s", stringPointerValue(src.Tab, "None")),
		fmt.Sprintf("URL: %s", stringPointerValue(src.CdpUrl, "None")),
		fmt.Sprintf("Idle: %t", src.Idle.Valid && src.Idle.Bool),
	}
	return strings.Join(lines, "\n")
}

func nullStringValue(valid bool, value, fallback string) string {
	if !valid || value == "" {
		return fallback
	}
	return value
}

func stringPointerValue(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "None"
	}
	return value.UTC().Format(time.RFC3339)
}
