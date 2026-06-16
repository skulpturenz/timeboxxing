//go:build windows

package idle

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	procGetLastInput = user32.NewProc("GetLastInputInfo")
	procGetTickCount = kernel32.NewProc("GetTickCount")
)

// lastInputInfo mirrors the LASTINPUTINFO Win32 structure.
type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

type windowsIdleDetector struct{}

// New returns the Windows IdleDetector.
func New(ctx context.Context) (IdleDetector, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return windowsIdleDetector{}, nil
}

func (windowsIdleDetector) SecondsSinceLastInput(ctx context.Context) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	var lii lastInputInfo
	lii.cbSize = uint32(unsafe.Sizeof(lii))
	ret, _, err := procGetLastInput.Call(uintptr(unsafe.Pointer(&lii)))
	if ret == 0 {
		return 0, fmt.Errorf("GetLastInputInfo: %w", err)
	}
	tick, _, _ := procGetTickCount.Call()
	idleMs := time.Duration(uint32(tick)-lii.dwTime) * time.Millisecond
	if idleMs < 0 {
		idleMs = 0 // tick counter wrapped around
	}
	return idleMs.Seconds(), nil
}
