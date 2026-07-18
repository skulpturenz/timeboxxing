//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework CoreGraphics

#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>
#include <libproc.h>
#include <stdlib.h>

char *copyProcessPath(int pid) {
    char path[PROC_PIDPATHINFO_MAXSIZE];
    int length = proc_pidpath(pid, path, sizeof(path));
    if (length <= 0) return NULL;
    return strdup(path);
}

// getActiveWindowAX queries the system-wide AX element for the truly focused
// application and its focused window. Works reliably from CLI processes.
//
// Return values: 0=success, 1=permission denied (AX not granted), 2=other error.
// On success, outAppName, outAppIdentifier, outAppPath, outWinTitle are heap-allocated (caller must free).
// outPid is set to the focused process PID.
int getActiveWindowAX(char **outAppName, char **outAppIdentifier, char **outAppPath, int *outPid, char **outWinTitle) {
    *outAppName       = NULL;
    *outAppIdentifier = NULL;
    *outAppPath       = NULL;
    *outPid           = -1;
    *outWinTitle      = NULL;

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
                if (app.bundleIdentifier) {
                    const char *s = [app.bundleIdentifier UTF8String];
                    if (s) *outAppIdentifier = strdup(s);
                }
                NSURL *url = app.bundleURL;
                if (url) {
                    const char *s = [[url path] UTF8String];
                    if (s) *outAppPath = strdup(s);
                }
                if (!*outAppPath) {
                    *outAppPath = copyProcessPath((int)pid);
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
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/permission"
)

type darwinTracker struct {
	cfg         Config
	mu          sync.Mutex
	axGranted   bool
	axLastCheck time.Time
}

// New returns the macOS Tracker implementation.
func New(ctx context.Context, cfg Config) (Tracker, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t := &darwinTracker{cfg: cfg}
	if cfg.PromptPermissions {
		C.isAXTrustedWithPrompt()
	}
	t.axGranted = C.isAXTrusted() == 1
	t.axLastCheck = time.Now()
	return t, nil
}

func (t *darwinTracker) Poll(ctx context.Context) (WindowInfo, error) {
	if err := ctx.Err(); err != nil {
		return WindowInfo{}, err
	}
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
		var nameC, identifierC, pathC, titleC *C.char
		var pid C.int
		rc := C.getActiveWindowAX(&nameC, &identifierC, &pathC, &pid, &titleC)

		if rc == 0 {
			focusedPID := int32(pid)
			info := WindowInfo{
				PID:         focusedPID,
				TitleSource: TitleSourceAX,
				Timestamp:   now,
			}
			appName := ""
			appIdentifier := ""
			appPath := ""
			if nameC != nil {
				appName = C.GoString(nameC)
				C.free(unsafe.Pointer(nameC))
			}
			if identifierC != nil {
				appIdentifier = C.GoString(identifierC)
				C.free(unsafe.Pointer(identifierC))
			}
			if pathC != nil {
				appPath = C.GoString(pathC)
				C.free(unsafe.Pointer(pathC))
			}
			if titleC != nil {
				info.WindowTitle = C.GoString(titleC)
				C.free(unsafe.Pointer(titleC))
			}
			identity := normalizeDarwinAppIdentity(appName, appIdentifier, appPath, processPath(focusedPID))
			info.AppName = identity.AppName
			info.AppIdentifier = identity.AppIdentifier
			info.AppPath = identity.AppPath
			info, _ = FinalizeWindowInfo(info)
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
	appName, identifier, pid, title := osascriptActiveWindow(ctx)
	if appName == "" && identifier == "" && pid <= 0 && title == "" {
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}, nil
	}
	identity := normalizeDarwinAppIdentity(appName, identifier, "", processPath(pid))
	info := WindowInfo{
		AppName:       identity.AppName,
		AppIdentifier: identity.AppIdentifier,
		AppPath:       identity.AppPath,
		PID:           pid,
		WindowTitle:   title,
		TitleSource:   TitleSourceOsascript,
		Timestamp:     now,
	}
	info, _ = FinalizeWindowInfo(info)
	return info, nil
}

func (t *darwinTracker) Permissions() []permission.Status {
	granted := C.isAXTrusted() == 1
	return []permission.Status{
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
func osascriptActiveWindow(ctx context.Context) (appName string, identifier string, pid int32, windowTitle string) {
	const script = `tell application "System Events"
		set p to first application process whose frontmost is true
		set appName to name of p
		set appPID to unix id of p
		set bundleID to ""
		try
			set bundleID to bundle identifier of p
		end try
		set winTitle to ""
		try
			set winTitle to name of front window of p
		end try
		return (appPID as text) & "|||" & appName & "|||" & bundleID & "|||" & winTitle
	end tell`

	out, err := exec.CommandContext(ctx, "osascript", "-e", script).Output()
	if err != nil {
		return "", "", 0, ""
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "|||", 4)
	if len(parts) < 2 {
		return "", "", 0, ""
	}
	if p, err := strconv.Atoi(parts[0]); err == nil {
		pid = int32(p)
	}
	appName = parts[1]
	if len(parts) >= 3 {
		identifier = parts[2]
	}
	if len(parts) == 4 {
		windowTitle = parts[3]
	}
	return
}

func pathFromBundleOrProcess(bundlePath string, pid int32) string {
	if strings.TrimSpace(bundlePath) != "" {
		return bundlePath
	}
	return processPath(pid)
}

type darwinAppIdentity struct {
	AppName       string
	AppIdentifier string
	AppPath       string
}

func normalizeDarwinAppIdentity(appName, identifier, bundlePath, executablePath string) darwinAppIdentity {
	if runtimeName, ok := darwinRuntimeExecutableName(executablePath); ok {
		return darwinAppIdentity{
			AppName:       runtimeName,
			AppIdentifier: runtimeName,
			AppPath:       strings.TrimSpace(executablePath),
		}
	}
	appPath := strings.TrimSpace(bundlePath)
	if appPath == "" {
		appPath = strings.TrimSpace(executablePath)
	}
	return darwinAppIdentity{
		AppName:       strings.TrimSpace(appName),
		AppIdentifier: strings.TrimSpace(identifier),
		AppPath:       appPath,
	}
}

func darwinRuntimeExecutableName(executablePath string) (string, bool) {
	name := strings.TrimSpace(filepath.Base(strings.TrimSpace(executablePath)))
	switch strings.ToLower(name) {
	case "java", "javaw":
		return strings.ToLower(name), true
	default:
		return "", false
	}
}

func processPath(pid int32) string {
	if pid <= 0 {
		return ""
	}
	pathC := C.copyProcessPath(C.int(pid))
	if pathC == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(pathC))
	return C.GoString(pathC)
}
