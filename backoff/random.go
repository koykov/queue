package backoff

import (
	"time"

	"github.com/koykov/queue/rng"
)

// Random (aka Full Jitter) applies to interval random value according formula `value/2 + random(value)`.
type Random struct {
	RNG rng.Interface
}

func (b *Random) Next(interval time.Duration, _ int) time.Duration {
	return interval/2 + time.Duration(b.RNG.Int63n(int64(interval)))
}
