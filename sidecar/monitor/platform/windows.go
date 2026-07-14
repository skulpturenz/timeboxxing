//go:build windows

package platform

import (
	"context"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGetForegroundWindow       = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW            = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId  = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess               = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle               = kernel32.NewProc("CloseHandle")
	procGetLastError              = kernel32.NewProc("GetLastError")
)

const (
	processQueryLimitedInformation = 0x1000
	maxProcessPathBufferSize       = 32768
)

// windowsExeToAppName normalises Windows executable names to human-readable app names.
var windowsExeToAppName = map[string]string{
	"chrome":          "Google Chrome",
	"chromium":        "Chromium",
	"msedge":          "Microsoft Edge",
	"firefox":         "Firefox",
	"code":            "Visual Studio Code",
	"devenv":          "Visual Studio",
	"windowsterminal": "Windows Terminal",
	"cmd":             "Command Prompt",
	"powershell":      "PowerShell",
	"pwsh":            "PowerShell",
	"notepad":         "Notepad",
	"notepad++":       "Notepad++",
	"explorer":        "File Explorer",
	"slack":           "Slack",
	"discord":         "Discord",
	"teams":           "Microsoft Teams",
	"outlook":         "Microsoft Outlook",
	"excel":           "Microsoft Excel",
	"winword":         "Microsoft Word",
	"powerpnt":        "Microsoft PowerPoint",
	"spotify":         "Spotify",
	"obsidian":        "Obsidian",
	"notion":          "Notion",
	"gitkraken":       "GitKraken",
	"sourcetree":      "Sourcetree",
	"sublime_text":    "Sublime Text",
}

type windowsTracker struct {
	cfg Config
}

// New returns the Windows Tracker implementation.
func New(ctx context.Context, cfg Config) (Tracker, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Kick off the background WinRT location provider; Poll reads its cache
	// non-blockingly. No-op beyond the first call.
	startWindowsLocation()
	return &windowsTracker{cfg: cfg}, nil
}

func (t *windowsTracker) Poll(ctx context.Context) (WindowInfo, error) {
	if err := ctx.Err(); err != nil {
		return WindowInfo{}, err
	}
	now := time.Now()

	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}, nil
	}

	// Window title.
	titleBuf := make([]uint16, 1024)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len(titleBuf)))
	title := windows.UTF16ToString(titleBuf)

	// PID.
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

	// Full executable path.
	appPath, appName := "", ""
	if pid != 0 {
		hProc, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
		if hProc != 0 {
			appPath = queryFullProcessImageName(hProc)
			procCloseHandle.Call(hProc)
		}
	}

	appIdentifier := ""
	if appPath != "" {
		appIdentifier, appName = windowsAppIdentity(appPath)
	}

	info := WindowInfo{
		AppName:       appName,
		AppIdentifier: appIdentifier,
		AppPath:       appPath,
		PID:           int32(pid),
		WindowTitle:   title,
		TitleSource:   TitleSourceWindowAPI,
		Timestamp:     now,
	}
	info, _ = FinalizeWindowInfo(info)
	return attachEnvironment(info), nil
}

func queryFullProcessImageName(processHandle uintptr) string {
	for bufferSize := uint32(windows.MAX_PATH); bufferSize <= maxProcessPathBufferSize; bufferSize *= 2 {
		pathBuf := make([]uint16, bufferSize)
		size := bufferSize
		ok, _, errno := procQueryFullProcessImageName.Call(
			processHandle,
			0,
			uintptr(unsafe.Pointer(&pathBuf[0])),
			uintptr(unsafe.Pointer(&size)),
		)
		if ok != 0 && size > 0 && size <= uint32(len(pathBuf)) {
			return windows.UTF16ToString(pathBuf[:size])
		}
		if errno != windows.ERROR_INSUFFICIENT_BUFFER {
			return ""
		}
	}
	return ""
}

func windowsAppIdentity(appPath string) (identifier string, appName string) {
	appPath = strings.TrimSpace(appPath)
	if appPath == "" {
		return "", ""
	}
	exe := strings.TrimSuffix(strings.ToLower(filepath.Base(appPath)), ".exe")
	if exe == "" {
		return "", ""
	}
	if runtimeName, ok := windowsRuntimeExecutableName(exe); ok {
		return runtimeName, runtimeName
	}
	if human, ok := windowsExeToAppName[exe]; ok {
		return exe, human
	}
	return exe, strings.ToUpper(exe[:1]) + exe[1:]
}

func windowsRuntimeExecutableName(exe string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(exe)) {
	case "java", "javaw":
		return strings.ToLower(strings.TrimSpace(exe)), true
	default:
		return "", false
	}
}

func (t *windowsTracker) Permissions() []PermissionStatus {
	// GetForegroundWindow requires no special permissions on Windows; location
	// does (WinRT Geolocator access grant).
	return []PermissionStatus{
		{Name: "None required", Granted: true, HowToGrant: ""},
		{
			Name:       "Location Services",
			Granted:    windowsLocationGranted(),
			HowToGrant: "Settings → Privacy & security → Location → enable location access for this app",
		},
	}
}
