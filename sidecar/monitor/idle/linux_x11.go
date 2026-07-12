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

type x11IdleDetector struct {
	mu   sync.Mutex
	conn *xgb.Conn
	root xproto.Drawable
}

// newX11IdleDetector returns an IdleDetector backed by the X11 MIT-SCREEN-SAVER
// extension, or nil when X11 (or the extension) is unavailable.
func newX11IdleDetector() *x11IdleDetector {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil
	}
	if err := screensaver.Init(conn); err != nil {
		conn.Close()
		return nil
	}
	setup := xproto.Setup(conn)
	root := xproto.Drawable(setup.DefaultScreen(conn).Root)
	return &x11IdleDetector{conn: conn, root: root}
}

func (d *x11IdleDetector) SecondsSinceLastInput(ctx context.Context) (float64, error) {
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
