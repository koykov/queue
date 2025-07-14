package backoff

import (
	"math/rand"
	"time"
)

// Random (aka Full Jitter) applies to interval random value according formula `value/2 + random(value)`.
type Random struct{}

func (Random) Next(interval time.Duration, _ int) time.Duration {
	return interval/2 + time.Duration(rand.Int63n(int64(interval)))
}
