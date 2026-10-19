package queue

import "time"

// Clock represents clock interface to get current time.
type Clock interface {
	Now() time.Time
}

type NativeClock struct{}

func (c NativeClock) Now() time.Time {
	return time.Now()
}
