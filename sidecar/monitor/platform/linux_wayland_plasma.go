//go:build linux

package platform

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/neurlang/wayland/wl"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/internal/wlproto/plasma"
)

// plasmaBackend tracks the active window on KDE Plasma/KWin using the native
// org_kde_plasma_window_management protocol. It is a fallback for KWin versions
// or configurations where wlr-foreign-toplevel-management is unavailable
// (modern KWin also implements the wlr protocol, which is preferred).
//
// Like the wlr backend, all window bookkeeping runs on the single event
// goroutine; only the snapshot is shared with Poll.
type plasmaBackend struct {
	runner *waylandRunner
	snap   waylandSnapshot

	windows map[*plasma.OrgKdeWindow]*plasmaWindowState
	active  *plasma.OrgKdeWindow
}

type plasmaWindowState struct {
	appID  string
	title  string
	active bool
}

func newPlasmaBackend(logger *slog.Logger) *plasmaBackend {
	b := &plasmaBackend{
		windows: map[*plasma.OrgKdeWindow]*plasmaWindowState{},
	}
	b.runner = newWaylandRunner(logger, "plasma", b.setup, b.snap.clear)
	b.runner.start()
	return b
}

func (b *plasmaBackend) setup(d *wl.Display, reg *wl.Registry, globals map[string]waylandGlobal) error {
	g, ok := globals[plasmaWindowManagementInterface]
	if !ok {
		return fmt.Errorf("%s not advertised", plasmaWindowManagementInterface)
	}
	b.windows = map[*plasma.OrgKdeWindow]*plasmaWindowState{}
	b.active = nil

	mgr := plasma.NewOrgKdeWindowManagement(d.Context())
	if err := reg.Bind(g.name, plasmaWindowManagementInterface, g.version, mgr); err != nil {
		return err
	}
	mgr.AddWindowWithUuidHandler(&plasmaMgmtListener{b: b, mgr: mgr})
	return nil
}

func (b *plasmaBackend) refresh() {
	if b.active == nil {
		b.snap.clear()
		return
	}
	st := b.windows[b.active]
	if st == nil || !st.active || (st.appID == "" && st.title == "") {
		b.snap.clear()
		return
	}
	b.snap.store(waylandWindowInfo(st.appID, st.title))
}

func (b *plasmaBackend) poll(now time.Time) (WindowInfo, error) {
	info, _ := b.snap.load(now)
	return info, nil
}

func (b *plasmaBackend) permissions() []PermissionStatus {
	return []PermissionStatus{
		{Name: "Wayland (plasma-window-management)", Granted: true, HowToGrant: ""},
	}
}

func (b *plasmaBackend) close() { b.runner.close() }

// plasmaMgmtListener receives new-window notifications from the management global.
type plasmaMgmtListener struct {
	b   *plasmaBackend
	mgr *plasma.OrgKdeWindowManagement
}

func (l *plasmaMgmtListener) HandleOrgKdeWindowManagementWindowWithUuid(ev plasma.OrgKdeWindowManagementWindowWithUuidEvent) {
	win, err := l.mgr.GetWindowByUuid(ev.Uuid)
	if err != nil || win == nil {
		return
	}
	l.b.windows[win] = &plasmaWindowState{}
	listener := &plasmaWindowListener{b: l.b, window: win}
	win.AddAppIdChangedHandler(listener)
	win.AddTitleChangedHandler(listener)
	win.AddStateChangedHandler(listener)
	win.AddUnmappedHandler(listener)
}

// plasmaWindowListener receives per-window events; one instance per window.
type plasmaWindowListener struct {
	b      *plasmaBackend
	window *plasma.OrgKdeWindow
}

func (l *plasmaWindowListener) HandleOrgKdeWindowAppIdChanged(ev plasma.OrgKdeWindowAppIdChangedEvent) {
	if st := l.b.windows[l.window]; st != nil {
		st.appID = ev.AppId
		l.b.refresh()
	}
}

func (l *plasmaWindowListener) HandleOrgKdeWindowTitleChanged(ev plasma.OrgKdeWindowTitleChangedEvent) {
	if st := l.b.windows[l.window]; st != nil {
		st.title = ev.Title
		l.b.refresh()
	}
}

func (l *plasmaWindowListener) HandleOrgKdeWindowStateChanged(ev plasma.OrgKdeWindowStateChangedEvent) {
	st := l.b.windows[l.window]
	if st == nil {
		return
	}
	st.active = ev.Flags&plasma.OrgKdeWindowManagementStateActive != 0
	if st.active {
		l.b.active = l.window
	} else if l.b.active == l.window {
		l.b.active = nil
	}
	l.b.refresh()
}

func (l *plasmaWindowListener) HandleOrgKdeWindowUnmapped(plasma.OrgKdeWindowUnmappedEvent) {
	if l.b.active == l.window {
		l.b.active = nil
	}
	delete(l.b.windows, l.window)
	l.b.refresh()
}
