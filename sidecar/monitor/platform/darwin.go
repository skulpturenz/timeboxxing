//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework CoreGraphics

#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>
#include <stdlib.h>

// getActiveWindowAX queries the system-wide AX element for the truly focused
// application and its focused window. Works reliably from CLI processes.
//
// Return values: 0=success, 1=permission denied (AX not granted), 2=other error.
// On success, outAppName, outAppPath, outWinTitle are heap-allocated (caller must free).
// outPid is set to the focused process PID.
int getActiveWindowAX(char **outAppName, char **outAppPath, int *outPid, char **outWinTitle) {
    *outAppName  = NULL;
    *outAppPath  = NULL;
    *outPid      = -1;
    *outWinTitle = NULL;

    // kAXFocusedApplicationAttribute on the system-wide element gives the truly
    // focused app, unlike NSWorkspace.frontmostApplication which is unreliable
    // from CLI/background processes.
    AXUIElementRef sysElem = AXUIElementCreateSystemWide();
    if (!sysElem) return 2;

    CFTypeRef focusedApp = NULL;
    AXError err = AXUIElementCopyAttributeValue(
        sysElem, kAXFocusedApplicationAttribute, &focusedApp);
    CFRelease(sysElem);

    if (err == kAXErrorAPIDisabled) return 1; // Accessibility not granted
    if (err != kAXErrorSuccess || !focusedApp) return 2;

    // Derive PID from the AX element.
    pid_t pid = -1;
    AXUIElementGetPid((AXUIElementRef)focusedApp, &pid);
    *outPid = (int)pid;

    // Use NSRunningApplication to get the human-readable name and bundle path.
    if (pid > 0) {
        @autoreleasepool {
            NSRunningApplication *app =
                [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
            if (app) {
                if (app.localizedName) {
                    const char *s = [app.localizedName UTF8String];
                    if (s) *outAppName = strdup(s);
                }
                NSURL *url = app.bundleURL;
                if (url) {
                    const char *s = [[url path] UTF8String];
                    if (s) *outAppPath = strdup(s);
                }
            }
        }
    }

    // Get the focused window title from the focused application element.
    CFTypeRef focusedWindow = NULL;
    err = AXUIElementCopyAttributeValue(
        (AXUIElementRef)focusedApp, kAXFocusedWindowAttribute, &focusedWindow);
    CFRelease(focusedApp);

    if (err == kAXErrorSuccess && focusedWindow) {
        CFTypeRef titleVal = NULL;
        if (AXUIElementCopyAttributeValue(
                (AXUIElementRef)focusedWindow, kAXTitleAttribute, &titleVal) == kAXErrorSuccess
            && titleVal
            && CFGetTypeID(titleVal) == CFStringGetTypeID()) {
            const char *s = [(__bridge NSString*)titleVal UTF8String];
            if (s) *outWinTitle = strdup(s);
        }
        if (titleVal) CFRelease(titleVal);
        CFRelease(focusedWindow);
    }

    return 0;
}

// isAXTrusted returns 1 if this process has Accessibility permission.
int isAXTrusted(void) {
    return AXIsProcessTrusted() ? 1 : 0;
}

// isAXTrustedWithPrompt returns 1 if trusted, and shows the macOS permission dialog if not.
int isAXTrustedWithPrompt(void) {
    NSDictionary *opts = @{(__bridge NSString*)kAXTrustedCheckOptionPrompt: @YES};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)opts) ? 1 : 0;
}
*/
import "C"

import (
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type darwinTracker struct {
	cfg         Config
	mu          sync.Mutex
	axGranted   bool
	axLastCheck time.Time
}

// New returns the macOS Tracker implementation.
func New(cfg Config) (Tracker, error) {
	t := &darwinTracker{cfg: cfg}
	if cfg.PromptPermissions {
		C.isAXTrustedWithPrompt()
	}
	t.axGranted = C.isAXTrusted() == 1
	t.axLastCheck = time.Now()
	return t, nil
}

func (t *darwinTracker) Poll() (WindowInfo, error) {
	now := time.Now()

	// Re-check AX permission every 60 s so permission grants take effect without restart.
	t.mu.Lock()
	if time.Since(t.axLastCheck) > 60*time.Second {
		t.axGranted = C.isAXTrusted() == 1
		t.axLastCheck = now
	}
	axGranted := t.axGranted
	t.mu.Unlock()

	if axGranted {
		var nameC, pathC, titleC *C.char
		var pid C.int
		rc := C.getActiveWindowAX(&nameC, &pathC, &pid, &titleC)

		if rc == 0 {
			info := WindowInfo{
				PID:         int32(pid),
				TitleSource: TitleSourceAX,
				Timestamp:   now,
			}
			if nameC != nil {
				info.AppName = C.GoString(nameC)
				C.free(unsafe.Pointer(nameC))
			}
			if pathC != nil {
				info.AppPath = C.GoString(pathC)
				C.free(unsafe.Pointer(pathC))
			}
			if titleC != nil {
				info.WindowTitle = C.GoString(titleC)
				C.free(unsafe.Pointer(titleC))
			}
			return info, nil
		}

		if rc == 1 {
			// Permission was revoked while running — update cache.
			t.mu.Lock()
			t.axGranted = false
			t.mu.Unlock()
		}
		// rc==2: transient AX failure — fall through to osascript.
	}

	// Fallback: osascript. Runs in its own process so it always sees the true
	// frontmost application regardless of how this CLI binary was launched.
	appName, pid, title := osascriptActiveWindow()
	if appName == "" {
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}, nil
	}
	return WindowInfo{
		AppName:     appName,
		PID:         pid,
		WindowTitle: title,
		TitleSource: TitleSourceOsascript,
		Timestamp:   now,
	}, nil
}

func (t *darwinTracker) Permissions() []PermissionStatus {
	granted := C.isAXTrusted() == 1
	return []PermissionStatus{
		{
			Name:       "Accessibility",
			Granted:    granted,
			HowToGrant: "System Settings → Privacy & Security → Accessibility → enable this app",
		},
	}
}

// osascriptActiveWindow queries System Events for the frontmost process name,
// its PID, and the title of its frontmost window — all in one subprocess call.
// This is the reliable fallback when the AX API is not available; osascript
// runs in its own process with a proper window-server connection.
func osascriptActiveWindow() (appName string, pid int32, windowTitle string) {
	const script = `tell application "System Events"
		set p to first application process whose frontmost is true
		set appName to name of p
		set appPID to unix id of p
		set winTitle to ""
		try
			set winTitle to name of front window of p
		end try
		return (appPID as text) & "|||" & appName & "|||" & winTitle
	end tell`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", 0, ""
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "|||", 3)
	if len(parts) < 2 {
		return "", 0, ""
	}
	if p, err := strconv.Atoi(parts[0]); err == nil {
		pid = int32(p)
	}
	appName = parts[1]
	if len(parts) == 3 {
		windowTitle = parts[2]
	}
	return
}
