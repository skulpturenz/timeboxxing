//go:build linux

package platform

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/neurlang/wayland/wl"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/internal/wlproto/wlr"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/permission"
)

// wlrBackend tracks the active toplevel on wlroots-based compositors (Sway,
// Hyprland, river, Wayfire, niri, COSMIC, labwc/phoc/cage) and KWin using the
// wlr-foreign-toplevel-management protocol. The handle whose state includes the
// "activated" flag is the foreground window.
//
// All handle bookkeeping happens on the runner's single event goroutine, so the
// handles map and active pointer need no locking; only the snapshot is shared
// with Poll.
type wlrBackend struct {
	runner *waylandRunner
	snap   waylandSnapshot

	handles map[*wlr.ZwlrForeignToplevelHandleV1]*wlrToplevelState
	active  *wlr.ZwlrForeignToplevelHandleV1
}

type wlrToplevelState struct {
	appID     string
	title     string
	activated bool
}

func newWlrBackend(logger *slog.Logger) *wlrBackend {
	b := &wlrBackend{
		handles: map[*wlr.ZwlrForeignToplevelHandleV1]*wlrToplevelState{},
	}
	b.runner = newWaylandRunner(logger, "wlr", b.setup, b.snap.clear)
	b.runner.start()
	return b
}

func (b *wlrBackend) setup(d *wl.Display, reg *wl.Registry, globals map[string]waylandGlobal) error {
	g, ok := globals[wlrForeignToplevelManagerInterface]
	if !ok {
		return fmt.Errorf("%s not advertised", wlrForeignToplevelManagerInterface)
	}
	// Reset per-connection state.
	b.handles = map[*wlr.ZwlrForeignToplevelHandleV1]*wlrToplevelState{}
	b.active = nil

	version := g.version
	if version > 3 {
		version = 3 // we understand protocol version 3
	}
	mgr := wlr.NewZwlrForeignToplevelManagerV1(d.Context())
	if err := reg.Bind(g.name, wlrForeignToplevelManagerInterface, version, mgr); err != nil {
		return err
	}
	mgr.AddToplevelHandler(b)
	return nil
}

// HandleZwlrForeignToplevelManagerV1Toplevel fires when a new toplevel appears.
func (b *wlrBackend) HandleZwlrForeignToplevelManagerV1Toplevel(ev wlr.ZwlrForeignToplevelManagerV1ToplevelEvent) {
	h := ev.Toplevel
	if h == nil {
		return
	}
	b.handles[h] = &wlrToplevelState{}
	l := &wlrHandleListener{b: b, handle: h}
	h.AddAppIdHandler(l)
	h.AddTitleHandler(l)
	h.AddStateHandler(l)
	h.AddDoneHandler(l)
	h.AddClosedHandler(l)
}

// refresh recomputes the cached foreground window from the active handle.
func (b *wlrBackend) refresh() {
	if b.active == nil {
		b.snap.clear()
		return
	}
	st := b.handles[b.active]
	if st == nil || !st.activated || (st.appID == "" && st.title == "") {
		b.snap.clear()
		return
	}
	b.snap.store(waylandWindowInfo(st.appID, st.title))
}

func (b *wlrBackend) poll(now time.Time) (WindowInfo, error) {
	info, _ := b.snap.load(now)
	return info, nil
}

func (b *wlrBackend) permissions() []permission.Status {
	return []permission.Status{
		{Name: "Wayland (wlr-foreign-toplevel-management)", Granted: true, HowToGrant: ""},
	}
}

func (b *wlrBackend) close() { b.runner.close() }

// wlrHandleListener receives per-toplevel events. One instance is created per
// handle so callbacks know which toplevel they belong to (the event structs
// themselves don't carry the handle).
type wlrHandleListener struct {
	b      *wlrBackend
	handle *wlr.ZwlrForeignToplevelHandleV1
}

func (l *wlrHandleListener) HandleZwlrForeignToplevelHandleV1AppId(ev wlr.ZwlrForeignToplevelHandleV1AppIdEvent) {
	if st := l.b.handles[l.handle]; st != nil {
		st.appID = ev.AppId
	}
}

func (l *wlrHandleListener) HandleZwlrForeignToplevelHandleV1Title(ev wlr.ZwlrForeignToplevelHandleV1TitleEvent) {
	if st := l.b.handles[l.handle]; st != nil {
		st.title = ev.Title
	}
}

func (l *wlrHandleListener) HandleZwlrForeignToplevelHandleV1State(ev wlr.ZwlrForeignToplevelHandleV1StateEvent) {
	st := l.b.handles[l.handle]
	if st == nil {
		return
	}
	st.activated = false
	for _, s := range ev.State {
		if s == wlr.ZwlrForeignToplevelHandleV1StateActivated {
			st.activated = true
		}
	}
	if st.activated {
		l.b.active = l.handle
	}
}

// HandleZwlrForeignToplevelHandleV1Done marks the end of an atomic update batch;
// commit the recomputed foreground window.
func (l *wlrHandleListener) HandleZwlrForeignToplevelHandleV1Done(wlr.ZwlrForeignToplevelHandleV1DoneEvent) {
	l.b.refresh()
}

func (l *wlrHandleListener) HandleZwlrForeignToplevelHandleV1Closed(wlr.ZwlrForeignToplevelHandleV1ClosedEvent) {
	if l.b.active == l.handle {
		l.b.active = nil
	}
	delete(l.b.handles, l.handle)
	l.b.refresh()
}
