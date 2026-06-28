package platform

import (
	"context"
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
