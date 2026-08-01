package utils

import "time"

type TimeSpan [2]time.Time

func (s TimeSpan) Between(r TimeSpan) bool {
	return !s[0].Before(r[0]) && !s[1].After(r[1])
}

// Overlaps reports whether s intersects r. The test is half-open and strict, so a span that merely
// touches a bound — and a zero-length one — is outside.
func (s TimeSpan) Overlaps(r TimeSpan) bool {
	return s[1].After(r[0]) && s[0].Before(r[1])
}

// Clip narrows s to the part of it falling inside r, reporting false when nothing does.
func (s TimeSpan) Clip(r TimeSpan) (TimeSpan, bool) {
	if s[0].Before(r[0]) {
		s[0] = r[0]
	}
	if s[1].After(r[1]) {
		s[1] = r[1]
	}

	if !s[1].After(s[0]) {
		return TimeSpan{}, false
	}

	return s, true
}

func (s TimeSpan) Duration() time.Duration {
	if !s[1].After(s[0]) {
		return 0
	}

	return s[1].Sub(s[0])
}

// ClippedDuration is the length of s once narrowed to r, and zero when it falls outside.
func (s TimeSpan) ClippedDuration(r TimeSpan) time.Duration {
	clipped, ok := s.Clip(r)
	if !ok {
		return 0
	}

	return clipped.Duration()
}
