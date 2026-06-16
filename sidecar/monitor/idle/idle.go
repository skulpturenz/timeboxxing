package idle

import "context"

// IdleDetector reports how long the user has been inactive.
type IdleDetector interface {
	// SecondsSinceLastInput returns the elapsed seconds since the last
	// keyboard, mouse, or touch event. Returns an error only on
	// non-recoverable failures (e.g., API unavailable).
	SecondsSinceLastInput(ctx context.Context) (float64, error)
}

// nopDetector is returned when idle detection is disabled or unsupported.
type nopDetector struct{}

func (nopDetector) SecondsSinceLastInput(context.Context) (float64, error) { return 0, nil }

// Nop returns a no-op IdleDetector that always reports zero idle time.
// Use when --no-idle is passed or the platform implementation is unavailable.
func Nop() IdleDetector { return nopDetector{} }
