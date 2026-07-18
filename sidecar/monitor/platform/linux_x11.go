//go:build linux

package platform

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/permission"
)

// x11Backend detects the active window on an X11 session (or XWayland) using
// EWMH properties read over a raw XGB connection.
type x11Backend struct {
	mu   sync.Mutex
	conn *xgb.Conn

	// Cached X11 atoms – interned once after connection.
	root        xproto.Window
	atomActive  xproto.Atom // _NET_ACTIVE_WINDOW
	atomWMName  xproto.Atom // _NET_WM_NAME (UTF-8)
	atomWMNameL xproto.Atom // WM_NAME (legacy Latin-1)
	atomWMClass xproto.Atom // WM_CLASS
	atomWMPID   xproto.Atom // _NET_WM_PID
	atomUTF8    xproto.Atom // UTF8_STRING
	atomsReady  bool
}

// newX11Backend opens an X11 connection. It returns nil when no X11 display is
// reachable, letting the caller fall back to another backend.
func newX11Backend() *x11Backend {
	b := &x11Backend{}
	if err := b.ensureConn(); err != nil {
		return nil
	}
	return b
}

func (b *x11Backend) poll(now time.Time) (WindowInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.ensureConn(); err != nil {
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone},
			fmt.Errorf("X11 connection unavailable: %w", err)
	}

	winID, err := b.getActiveWindowID()
	if err != nil {
		b.conn = nil // force reconnect on next poll
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone},
			fmt.Errorf("_NET_ACTIVE_WINDOW: %w", err)
	}

	title := b.getWindowTitle(winID)
	windowClass := b.getWindowClass(winID)
	pid := b.getWindowPID(winID)

	appName, appPath := "", ""
	if pid > 0 {
		appName = procComm(pid)
		appPath = procExe(pid)
	}
	identity := normalizeLinuxAppIdentity(appName, appPath, windowClass, title)

	return WindowInfo{
		AppName:       identity.AppName,
		AppIdentifier: identity.AppIdentifier,
		AppPath:       identity.AppPath,
		PID:           int32(pid),
		WindowTitle:   title,
		TitleSource:   TitleSourceWindowAPI,
		Timestamp:     now,
	}, nil
}

func (b *x11Backend) permissions() []permission.Status {
	// X11 window tracking requires no special permissions.
	return []permission.Status{
		{Name: "X11 Display", Granted: os.Getenv("DISPLAY") != "", HowToGrant: "set the DISPLAY environment variable (e.g. DISPLAY=:0)"},
	}
}

func (b *x11Backend) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn != nil {
		b.conn.Close()
		b.conn = nil
	}
}

// ensureConn opens (or re-opens) the X11 connection and interns atoms.
// Must be called with b.mu held.
func (b *x11Backend) ensureConn() error {
	if b.conn != nil {
		return nil
	}
	conn, err := xgb.NewConn()
	if err != nil {
		return err
	}
	b.conn = conn
	b.atomsReady = false
	setup := xproto.Setup(conn)
	b.root = setup.DefaultScreen(conn).Root
	return b.internAtoms()
}

func (b *x11Backend) internAtoms() error {
	type atomReq struct {
		name string
		dest *xproto.Atom
	}
	reqs := []atomReq{
		{"_NET_ACTIVE_WINDOW", &b.atomActive},
		{"_NET_WM_NAME", &b.atomWMName},
		{"WM_NAME", &b.atomWMNameL},
		{"WM_CLASS", &b.atomWMClass},
		{"_NET_WM_PID", &b.atomWMPID},
		{"UTF8_STRING", &b.atomUTF8},
	}
	// Send all intern-atom requests first, then collect replies (pipeline).
	cookies := make([]xproto.InternAtomCookie, len(reqs))
	for i, r := range reqs {
		cookies[i] = xproto.InternAtom(b.conn, true, uint16(len(r.name)), r.name)
	}
	for i, r := range reqs {
		reply, err := cookies[i].Reply()
		if err != nil {
			return fmt.Errorf("intern atom %s: %w", r.name, err)
		}
		*r.dest = reply.Atom
	}
	b.atomsReady = true
	return nil
}

func (b *x11Backend) getActiveWindowID() (xproto.Window, error) {
	reply, err := xproto.GetProperty(
		b.conn, false, b.root, b.atomActive,
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

func (b *x11Backend) getWindowTitle(win xproto.Window) string {
	// Try _NET_WM_NAME (UTF-8) first.
	reply, err := xproto.GetProperty(
		b.conn, false, win, b.atomWMName,
		b.atomUTF8, 0, (1<<32)-1,
	).Reply()
	if err == nil && len(reply.Value) > 0 {
		return string(reply.Value)
	}
	// Fallback to legacy WM_NAME (Latin-1 / compound text).
	reply, err = xproto.GetProperty(
		b.conn, false, win, b.atomWMNameL,
		xproto.GetPropertyTypeAny, 0, (1<<32)-1,
	).Reply()
	if err == nil && len(reply.Value) > 0 {
		// Strip non-printable bytes and return as string.
		return sanitiseLatin1(reply.Value)
	}
	return ""
}

func (b *x11Backend) getWindowClass(win xproto.Window) string {
	reply, err := xproto.GetProperty(
		b.conn, false, win, b.atomWMClass,
		xproto.GetPropertyTypeAny, 0, (1<<32)-1,
	).Reply()
	if err != nil || len(reply.Value) == 0 {
		return ""
	}
	return parseWMClass(reply.Value)
}

func (b *x11Backend) getWindowPID(win xproto.Window) uint32 {
	reply, err := xproto.GetProperty(
		b.conn, false, win, b.atomWMPID,
		xproto.GetPropertyTypeAny, 0, 1,
	).Reply()
	if err != nil || len(reply.Value) < 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(reply.Value)
}
