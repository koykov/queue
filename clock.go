package blqueue

import "time"

// Clock represents clock interface to get current time.
type Clock interface {
	Now() time.Time
}

type nativeClock struct{}

func (c nativeClock) Now() time.Time {
	return time.Now()
}
