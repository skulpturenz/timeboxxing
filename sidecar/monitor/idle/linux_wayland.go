//go:build linux

package idle

import (
	"context"
	"sync"
	"time"

	"github.com/neurlang/wayland/wl"
	"github.com/neurlang/wayland/wlclient"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/internal/wlproto/ext"
)

const (
	extIdleNotifierInterface = "ext_idle_notifier_v1"
	wlSeatInterface          = "wl_seat"
	// waylandIdleTimeout is how long without input before the compositor reports
	// idle. It doubles as the resolution of SecondsSinceLastInput; the session
	// manager only compares against a multi-minute threshold, so 1s is ample.
	waylandIdleTimeout = time.Second
	waylandIdleBackoff = 2 * time.Second
)

// waylandIdleDetector reports idle time using the ext-idle-notify-v1 protocol.
// The compositor pushes idled/resumed events for a notification created with a
// short timeout; the detector tracks when idle began and derives elapsed idle
// seconds from it. It owns a persistent connection with a reconnect loop for
// the lifetime of the process (the IdleDetector interface has no Close).
type waylandIdleDetector struct {
	mu        sync.Mutex
	idleSince time.Time // zero when the user is active (or state unknown)
}

// newWaylandIdleDetector returns a detector if the compositor advertises
// ext-idle-notify-v1 and a wl_seat, or nil to let the caller fall back.
func newWaylandIdleDetector() *waylandIdleDetector {
	available, err := waylandIdleAvailable()
	if err != nil || !available {
		return nil
	}
	d := &waylandIdleDetector{}
	go d.loop()
	return d
}

func (d *waylandIdleDetector) SecondsSinceLastInput(ctx context.Context) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.idleSince.IsZero() {
		return 0, nil
	}
	// The idled event fires waylandIdleTimeout after the last input, so add it
	// back to approximate true seconds-since-input.
	return waylandIdleTimeout.Seconds() + time.Since(d.idleSince).Seconds(), nil
}

// HandleIdleNotificationV1Idled fires once the seat has been idle for the
// configured timeout.
func (d *waylandIdleDetector) HandleIdleNotificationV1Idled(ext.IdleNotificationV1IdledEvent) {
	d.mu.Lock()
	d.idleSince = time.Now()
	d.mu.Unlock()
}

// HandleIdleNotificationV1Resumed fires on the next input after idle.
func (d *waylandIdleDetector) HandleIdleNotificationV1Resumed(ext.IdleNotificationV1ResumedEvent) {
	d.mu.Lock()
	d.idleSince = time.Time{}
	d.mu.Unlock()
}

func (d *waylandIdleDetector) loop() {
	for {
		_ = d.serve()
		// Connection dropped: treat idle state as unknown (active) and retry.
		d.mu.Lock()
		d.idleSince = time.Time{}
		d.mu.Unlock()
		time.Sleep(waylandIdleBackoff)
	}
}

func (d *waylandIdleDetector) serve() error {
	display, err := wl.Connect("")
	if err != nil {
		return err
	}
	defer wlclient.DisplayDisconnect(display)

	reg, err := display.GetRegistry()
	if err != nil {
		return err
	}
	collector := &idleGlobalCollector{globals: map[string]idleGlobal{}}
	wlclient.RegistryAddListener(reg, collector)
	if err := wlclient.DisplayRoundtrip(display); err != nil {
		return err
	}

	notifierG, hasNotifier := collector.globals[extIdleNotifierInterface]
	seatG, hasSeat := collector.globals[wlSeatInterface]
	if !hasNotifier || !hasSeat {
		return errNoWaylandIdle
	}

	notifier := ext.NewIdleNotifierV1(display.Context())
	if err := reg.Bind(notifierG.name, extIdleNotifierInterface, notifierG.version, notifier); err != nil {
		return err
	}
	seat := wlclient.RegistryBindSeatInterface(reg, seatG.name, seatG.version)

	notification, err := notifier.GetIdleNotification(uint32(waylandIdleTimeout/time.Millisecond), seat)
	if err != nil {
		return err
	}
	notification.AddIdledHandler(d)
	notification.AddResumedHandler(d)
	if err := wlclient.DisplayRoundtrip(display); err != nil {
		return err
	}

	for {
		if err := wlclient.DisplayDispatch(display); err != nil {
			return err
		}
	}
}

// waylandIdleAvailable probes the compositor for the required globals.
func waylandIdleAvailable() (bool, error) {
	display, err := wl.Connect("")
	if err != nil {
		return false, err
	}
	defer wlclient.DisplayDisconnect(display)

	reg, err := display.GetRegistry()
	if err != nil {
		return false, err
	}
	collector := &idleGlobalCollector{globals: map[string]idleGlobal{}}
	wlclient.RegistryAddListener(reg, collector)
	if err := wlclient.DisplayRoundtrip(display); err != nil {
		return false, err
	}
	_, hasNotifier := collector.globals[extIdleNotifierInterface]
	_, hasSeat := collector.globals[wlSeatInterface]
	return hasNotifier && hasSeat, nil
}

var errNoWaylandIdle = errWaylandIdle("ext-idle-notify-v1 or wl_seat not advertised")

type errWaylandIdle string

func (e errWaylandIdle) Error() string { return string(e) }

type idleGlobal struct {
	name    uint32
	version uint32
}

type idleGlobalCollector struct {
	globals map[string]idleGlobal
}

func (c *idleGlobalCollector) HandleRegistryGlobal(ev wl.RegistryGlobalEvent) {
	c.globals[ev.Interface] = idleGlobal{name: ev.Name, version: ev.Version}
}

func (c *idleGlobalCollector) HandleRegistryGlobalRemove(wl.RegistryGlobalRemoveEvent) {}
