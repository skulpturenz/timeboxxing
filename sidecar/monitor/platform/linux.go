//go:build linux

package platform

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

type linuxTracker struct {
	cfg  Config
	mu   sync.Mutex
	conn *xgb.Conn

	// Cached X11 atoms – interned once after connection.
	root        xproto.Window
	atomActive  xproto.Atom // _NET_ACTIVE_WINDOW
	atomWMName  xproto.Atom // _NET_WM_NAME (UTF-8)
	atomWMNameL xproto.Atom // WM_NAME (legacy Latin-1)
	atomWMPID   xproto.Atom // _NET_WM_PID
	atomUTF8    xproto.Atom // UTF8_STRING
	atomsReady  bool
}

// New returns the Linux Tracker implementation.
func New(ctx context.Context, cfg Config) (Tracker, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t := &linuxTracker{cfg: cfg}
	// Attempt an initial connection; non-fatal if it fails (will retry on first Poll).
	_ = t.ensureConn()
	return t, nil
}

func (t *linuxTracker) Poll(ctx context.Context) (WindowInfo, error) {
	if err := ctx.Err(); err != nil {
		return WindowInfo{}, err
	}
	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.ensureConn(); err != nil {
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone},
			fmt.Errorf("X11 connection unavailable: %w", err)
	}

	winID, err := t.getActiveWindowID()
	if err != nil {
		t.conn = nil // force reconnect on next poll
		return t.waylandFallback(now), nil
	}

	title := t.getWindowTitle(winID)
	pid := t.getWindowPID(winID)

	appName, appPath := "", ""
	if pid > 0 {
		appName = procComm(pid)
		appPath = procExe(pid)
	}
	if appName == "" && title != "" {
		// Best-effort: use the window title as the app name.
		appName = title
	}

	return WindowInfo{
		AppName:     appName,
		AppPath:     appPath,
		PID:         int32(pid),
		WindowTitle: title,
		TitleSource: TitleSourceWindowAPI,
		Timestamp:   now,
	}, nil
}

func (t *linuxTracker) Permissions() []PermissionStatus {
	// X11 window tracking requires no special permissions.
	return []PermissionStatus{
		{Name: "X11 Display", Granted: os.Getenv("DISPLAY") != "", HowToGrant: "set the DISPLAY environment variable (e.g. DISPLAY=:0)"},
	}
}

// ensureConn opens (or re-opens) the X11 connection and interns atoms.
// Must be called with t.mu held.
func (t *linuxTracker) ensureConn() error {
	if t.conn != nil {
		return nil
	}
	conn, err := xgb.NewConn()
	if err != nil {
		return err
	}
	t.conn = conn
	t.atomsReady = false
	setup := xproto.Setup(conn)
	t.root = setup.DefaultScreen(conn).Root
	return t.internAtoms()
}

func (t *linuxTracker) internAtoms() error {
	type atomReq struct {
		name string
		dest *xproto.Atom
	}
	reqs := []atomReq{
		{"_NET_ACTIVE_WINDOW", &t.atomActive},
		{"_NET_WM_NAME", &t.atomWMName},
		{"WM_NAME", &t.atomWMNameL},
		{"_NET_WM_PID", &t.atomWMPID},
		{"UTF8_STRING", &t.atomUTF8},
	}
	// Send all intern-atom requests first, then collect replies (pipeline).
	cookies := make([]xproto.InternAtomCookie, len(reqs))
	for i, r := range reqs {
		cookies[i] = xproto.InternAtom(t.conn, true, uint16(len(r.name)), r.name)
	}
	for i, r := range reqs {
		reply, err := cookies[i].Reply()
		if err != nil {
			return fmt.Errorf("intern atom %s: %w", r.name, err)
		}
		*r.dest = reply.Atom
	}
	t.atomsReady = true
	return nil
}

func (t *linuxTracker) getActiveWindowID() (xproto.Window, error) {
	reply, err := xproto.GetProperty(
		t.conn, false, t.root, t.atomActive,
		xproto.GetPropertyTypeAny, 0, 1,
	).Reply()
	if err != nil {
		return 0, err
	}
	if len(reply.Value) < 4 {
		return 0, fmt.Errorf("_NET_ACTIVE_WINDOW: short reply")
	}
	return xproto.Window(binary.LittleEndian.Uint32(reply.Value)), nil
}

func (t *linuxTracker) getWindowTitle(win xproto.Window) string {
	// Try _NET_WM_NAME (UTF-8) first.
	reply, err := xproto.GetProperty(
		t.conn, false, win, t.atomWMName,
		t.atomUTF8, 0, (1<<32)-1,
	).Reply()
	if err == nil && len(reply.Value) > 0 {
		return string(reply.Value)
	}
	// Fallback to legacy WM_NAME (Latin-1 / compound text).
	reply, err = xproto.GetProperty(
		t.conn, false, win, t.atomWMNameL,
		xproto.GetPropertyTypeAny, 0, (1<<32)-1,
	).Reply()
	if err == nil && len(reply.Value) > 0 {
		// Strip non-printable bytes and return as string.
		return sanitiseLatin1(reply.Value)
	}
	return ""
}

func (t *linuxTracker) getWindowPID(win xproto.Window) uint32 {
	reply, err := xproto.GetProperty(
		t.conn, false, win, t.atomWMPID,
		xproto.GetPropertyTypeAny, 0, 1,
	).Reply()
	if err != nil || len(reply.Value) < 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(reply.Value)
}

// waylandFallback tries xdotool when X11 is unavailable.
func (t *linuxTracker) waylandFallback(now time.Time) WindowInfo {
	out, err := exec.Command("xdotool", "getactivewindow", "getwindowname").Output()
	info := WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}
	if err != nil {
		return info
	}
	title := strings.TrimSpace(string(out))
	info.WindowTitle = title
	info.AppName = title
	info.TitleSource = TitleSourceWindowAPI
	return info
}

// procComm reads the process name from /proc/<pid>/comm.
func procComm(pid uint32) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// procExe reads the executable path from /proc/<pid>/exe symlink.
func procExe(pid uint32) string {
	link, err := filepath.EvalSymlinks(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return ""
	}
	return link
}

// sanitiseLatin1 converts a Latin-1 byte slice to a printable UTF-8 string.
func sanitiseLatin1(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		if c >= 0x20 && c != 0x7f {
			sb.WriteByte(c)
		}
	}
	return sb.String()
}
