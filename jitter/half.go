package jitter

import (
	"math/rand"
	"time"
)

// Half returns half of original interval and random value of (0...half].
type Half struct{}

func (Half) Apply(interval time.Duration) time.Duration {
	half := interval / 2
	return half + time.Duration(rand.Int63n(int64(half)))
}
