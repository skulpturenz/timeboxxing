package utils

import "time"

type TimeSpan [2]time.Time

func (s TimeSpan) Between(r TimeSpan) bool {
	return !s[0].Before(r[0]) && !s[1].After(r[1])
}
