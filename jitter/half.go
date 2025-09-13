package jitter

import (
	"sync"
	"time"

	"github.com/koykov/queue/rng"
)

// Half returns half of original interval and random value of [0...half).
type Half struct {
	RNG  rng.Interface
	once sync.Once
}

func (j *Half) Apply(interval time.Duration) time.Duration {
	j.once.Do(func() {
		if j.RNG == nil {
			j.RNG = &rng.Pool{}
		}
	})
	half := interval / 2
	return half + time.Duration(j.RNG.Int63n(int64(half)))
}
