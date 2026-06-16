//go:build darwin

package idle

/*
#cgo LDFLAGS: -framework CoreGraphics

#include <CoreGraphics/CoreGraphics.h>

double secondsSinceLastInput(void) {
    // kCGAnyInputEventType covers keyboard, mouse buttons, mouse movement, scroll, touch.
    return CGEventSourceSecondsSinceLastEventType(
        kCGEventSourceStateCombinedSessionState,
        kCGAnyInputEventType);
}
*/
import "C"

import "context"

type darwinIdleDetector struct{}

// New returns the macOS IdleDetector.
func New(ctx context.Context) (IdleDetector, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return darwinIdleDetector{}, nil
}

func (darwinIdleDetector) SecondsSinceLastInput(ctx context.Context) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return float64(C.secondsSinceLastInput()), nil
}
