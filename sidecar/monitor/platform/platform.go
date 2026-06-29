package platform

import (
	"context"
	"strings"
	"time"
)

// TitleSource documents which OS API produced the WindowTitle.
type TitleSource string

const (
	TitleSourceAX        TitleSource = "ax"         // macOS Accessibility API
	TitleSourceOsascript TitleSource = "osascript"  // macOS AppleScript fallback
	TitleSourceWindowAPI TitleSource = "window_api" // Windows GetWindowText / Linux _NET_WM_NAME
	TitleSourceNone      TitleSource = "none"       // permission denied or unavailable
)

// WindowInfo is a single observation from one poll tick.
type WindowInfo struct {
	AppName       string
	AppIdentifier string
	AppPath       string
	PID           int32
	WindowTitle   string
	TitleSource   TitleSource
	Timestamp     time.Time
}

const UnknownAppName = "Unknown app"

// FinalizeWindowInfo normalizes a platform observation into a displayable
// foreground identity. It returns false only when the sample has no foreground
// signal at all.
func FinalizeWindowInfo(info WindowInfo) (WindowInfo, bool) {
	info.AppName = strings.TrimSpace(info.AppName)
	info.AppIdentifier = strings.TrimSpace(info.AppIdentifier)
	info.AppPath = strings.TrimSpace(info.AppPath)
	info.WindowTitle = strings.TrimSpace(info.WindowTitle)

	if info.AppName == "" {
		info.AppName = firstNonEmpty(
			displayNameFromPath(info.AppPath),
			displayNameFromIdentifier(info.AppIdentifier),
			info.WindowTitle,
		)
	}
	if info.AppName == "" && hasForegroundEvidence(info) {
		info.AppName = UnknownAppName
	}

	return info, info.AppName != ""
}

func hasForegroundEvidence(info WindowInfo) bool {
	return info.AppName != "" ||
		info.AppIdentifier != "" ||
		info.AppPath != "" ||
		info.WindowTitle != "" ||
		info.PID > 0 ||
		(info.TitleSource != "" && info.TitleSource != TitleSourceNone)
}

func displayNameFromPath(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), `/\`)
	if value == "" {
		return ""
	}
	separatorIndex := strings.LastIndexAny(value, `/\`)
	if separatorIndex >= 0 {
		value = value[separatorIndex+1:]
	}
	lower := strings.ToLower(value)
	for _, suffix := range []string{".app", ".exe"} {
		if strings.HasSuffix(lower, suffix) {
			return strings.TrimSpace(value[:len(value)-len(suffix)])
		}
	}
	return strings.TrimSpace(value)
}

func displayNameFromIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ".")
	for index := len(parts) - 1; index >= 0; index-- {
		if part := strings.TrimSpace(parts[index]); part != "" {
			return part
		}
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// PermissionStatus describes one OS permission required by this platform tracker.
type PermissionStatus struct {
	Name       string
	Granted    bool
	HowToGrant string
}

// PermissionError is returned when a required permission has not been granted.
type PermissionError struct {
	Permission string
	Detail     string
}

func (e *PermissionError) Error() string {
	return "permission required: " + e.Permission + ": " + e.Detail
}

// Config holds tunables passed from main into the platform layer.
type Config struct {
	// macOS: prefer AX API over osascript for window titles.
	PreferAX bool
	// macOS: trigger the system permission dialog at startup.
	PromptPermissions bool
}

// Tracker is the platform abstraction for active window detection.
// Each platform file implements New() returning a Tracker.
type Tracker interface {
	// Poll returns the current foreground window state.
	Poll(ctx context.Context) (WindowInfo, error)
	// Permissions returns the list of permissions this implementation requires
	// and whether each is currently granted.
	Permissions() []PermissionStatus
}
