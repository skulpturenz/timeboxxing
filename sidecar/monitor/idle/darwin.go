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

type darwinIdleDetector struct{}

// New returns the macOS IdleDetector.
func New() (IdleDetector, error) {
	return darwinIdleDetector{}, nil
}

func (darwinIdleDetector) SecondsSinceLastInput() (float64, error) {
	return float64(C.secondsSinceLastInput()), nil
}
