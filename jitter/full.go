package jitter

import (
	"math/rand"
	"time"
)

// Full returns random value of (0...interval].
type Full struct{}

func (Full) Apply(interval time.Duration) time.Duration {
	return time.Duration(rand.Int63n(int64(interval)))
}
