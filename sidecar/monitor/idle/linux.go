//go:build linux

package idle

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/screensaver"
	"github.com/BurntSushi/xgb/xproto"
)

type linuxIdleDetector struct {
	mu   sync.Mutex
	conn *xgb.Conn
	root xproto.Drawable
}

// New returns the Linux IdleDetector using the X11 MIT-SCREEN-SAVER extension.
// Falls back to a no-op detector when X11 is unavailable (e.g. pure Wayland).
func New(ctx context.Context) (IdleDetector, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn, err := xgb.NewConn()
	if err != nil {
		// No X11 display — return a no-op so the rest of the app still works.
		return Nop(), nil
	}
	if err := screensaver.Init(conn); err != nil {
		conn.Close()
		return Nop(), nil
	}
	setup := xproto.Setup(conn)
	root := xproto.Drawable(setup.DefaultScreen(conn).Root)
	return &linuxIdleDetector{conn: conn, root: root}, nil
}

func (d *linuxIdleDetector) SecondsSinceLastInput(ctx context.Context) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	info, err := screensaver.QueryInfo(d.conn, d.root).Reply()
	if err != nil {
		return 0, fmt.Errorf("screensaver.QueryInfo: %w", err)
	}
	idleMs := time.Duration(info.MsSinceUserInput) * time.Millisecond
	return idleMs.Seconds(), nil
}
