//go:build linux

package platform

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/neurlang/wayland/wl"
	"github.com/neurlang/wayland/wlclient"
)

// Wayland global interface names we care about for foreground detection.
const (
	wlrForeignToplevelManagerInterface = "zwlr_foreign_toplevel_manager_v1"
	plasmaWindowManagementInterface    = "org_kde_plasma_window_management"
)

// waylandGlobal is a compositor-advertised global (its registry name + version).
type waylandGlobal struct {
	name    uint32
	version uint32
}

// globalCollector records every advertised global during a registry roundtrip.
type globalCollector struct {
	globals map[string]waylandGlobal
}

func (c *globalCollector) HandleRegistryGlobal(ev wl.RegistryGlobalEvent) {
	c.globals[ev.Interface] = waylandGlobal{name: ev.Name, version: ev.Version}
}

func (c *globalCollector) HandleRegistryGlobalRemove(wl.RegistryGlobalRemoveEvent) {}

// probeWaylandGlobals connects briefly, lists the advertised globals, and
// disconnects. Used once at startup to pick a foreground backend.
func probeWaylandGlobals() (map[string]waylandGlobal, error) {
	d, err := wl.Connect("")
	if err != nil {
		return nil, err
	}
	defer wlclient.DisplayDisconnect(d)

	reg, err := d.GetRegistry()
	if err != nil {
		return nil, err
	}
	collector := &globalCollector{globals: map[string]waylandGlobal{}}
	wlclient.RegistryAddListener(reg, collector)
	if err := wlclient.DisplayRoundtrip(d); err != nil {
		return nil, err
	}
	return collector.globals, nil
}

// newWaylandForegroundBackend probes the compositor and returns a backend for
// the first supported foreground protocol, or nil if none is available.
func newWaylandForegroundBackend(cfg Config, logger *slog.Logger) linuxBackend {
	globals, err := probeWaylandGlobals()
	if err != nil {
		logger.Debug("wayland global probe failed", "error", err)
		return nil
	}
	if _, ok := globals[wlrForeignToplevelManagerInterface]; ok {
		logger.Info("using wlr-foreign-toplevel-management for Wayland foreground tracking")
		return newWlrBackend(logger)
	}
	if _, ok := globals[plasmaWindowManagementInterface]; ok {
		logger.Info("using KDE plasma-window-management for Wayland foreground tracking")
		return newPlasmaBackend(logger)
	}
	return nil
}

// waylandSnapshot is the mutex-guarded foreground state shared between a
// backend's event goroutine (writer) and Poll (reader).
type waylandSnapshot struct {
	mu   sync.Mutex
	info WindowInfo
	set  bool
}

func (s *waylandSnapshot) store(info WindowInfo) {
	s.mu.Lock()
	s.info = info
	s.set = true
	s.mu.Unlock()
}

func (s *waylandSnapshot) clear() {
	s.mu.Lock()
	s.info = WindowInfo{}
	s.set = false
	s.mu.Unlock()
}

func (s *waylandSnapshot) load(now time.Time) (WindowInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.set {
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}, false
	}
	info := s.info
	info.Timestamp = now
	return info, true
}

// waylandWindowInfo builds a WindowInfo from a Wayland app_id + title. The
// toplevel-management protocols expose no PID, so AppPath/PID stay empty; app
// identity is derived from the app_id (reverse-DNS aware) with a title fallback.
func waylandWindowInfo(appID, title string) WindowInfo {
	appID = strings.TrimSpace(appID)
	title = strings.TrimSpace(title)
	appName := displayNameFromIdentifier(appID)
	if appName == "" {
		appName = title
	}
	return WindowInfo{
		AppName:       appName,
		AppIdentifier: appID,
		WindowTitle:   title,
		TitleSource:   TitleSourceWindowAPI,
	}
}

// waylandRunner owns a Wayland connection and reconnect loop for an
// event-driven backend. It calls setup on each (re)connection to bind protocol
// globals and subscribe handlers, then dispatches events until the connection
// drops, retrying with a fixed backoff until close.
type waylandRunner struct {
	logger *slog.Logger
	label  string
	// setup binds protocol globals and subscribes handlers. It is invoked on
	// every (re)connection and must reset any per-connection state.
	setup func(d *wl.Display, reg *wl.Registry, globals map[string]waylandGlobal) error
	// onDisconnect is called after each connection drops so the backend can
	// clear cached state.
	onDisconnect func()

	mu      sync.Mutex
	display *wl.Display
	stopped bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

func newWaylandRunner(
	logger *slog.Logger,
	label string,
	setup func(*wl.Display, *wl.Registry, map[string]waylandGlobal) error,
	onDisconnect func(),
) *waylandRunner {
	return &waylandRunner{
		logger:       logger,
		label:        label,
		setup:        setup,
		onDisconnect: onDisconnect,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}
}

func (r *waylandRunner) start() { go r.loop() }

func (r *waylandRunner) loop() {
	defer close(r.doneCh)
	const backoff = 2 * time.Second
	for {
		r.mu.Lock()
		stopped := r.stopped
		r.mu.Unlock()
		if stopped {
			return
		}

		err := r.serve()
		if r.onDisconnect != nil {
			r.onDisconnect()
		}

		r.mu.Lock()
		stopped = r.stopped
		r.mu.Unlock()
		if stopped {
			return
		}
		if err != nil {
			r.logger.Debug("wayland connection dropped; reconnecting", "backend", r.label, "error", err)
		}
		select {
		case <-time.After(backoff):
		case <-r.stopCh:
			return
		}
	}
}

// serve runs one connection lifecycle: connect, bind, dispatch until error.
func (r *waylandRunner) serve() error {
	d, err := wl.Connect("")
	if err != nil {
		return err
	}
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		wlclient.DisplayDisconnect(d)
		return nil
	}
	r.display = d
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.display = nil
		r.mu.Unlock()
		wlclient.DisplayDisconnect(d)
	}()

	reg, err := d.GetRegistry()
	if err != nil {
		return err
	}
	collector := &globalCollector{globals: map[string]waylandGlobal{}}
	wlclient.RegistryAddListener(reg, collector)
	if err := wlclient.DisplayRoundtrip(d); err != nil {
		return err
	}
	if err := r.setup(d, reg, collector.globals); err != nil {
		return err
	}
	// Second roundtrip delivers the initial protocol events (existing
	// toplevels and their app_id/title/state).
	if err := wlclient.DisplayRoundtrip(d); err != nil {
		return err
	}
	for {
		if err := wlclient.DisplayDispatch(d); err != nil {
			return err
		}
	}
}

// close stops the reconnect loop and interrupts any blocking dispatch by
// closing the live connection, then waits for the goroutine to exit.
func (r *waylandRunner) close() {
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	r.stopped = true
	close(r.stopCh)
	d := r.display
	r.mu.Unlock()
	if d != nil {
		wlclient.DisplayDisconnect(d)
	}
	<-r.doneCh
}
