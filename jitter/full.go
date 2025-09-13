package jitter

import (
	"sync"
	"time"

	"github.com/koykov/queue/rng"
)

// Full returns random value of [0...interval).
type Full struct {
	RNG  rng.Interface
	once sync.Once
}

func (j *Full) Apply(interval time.Duration) time.Duration {
	j.once.Do(func() {
		if j.RNG == nil {
			j.RNG = &rng.Pool{}
		}
	})
	return time.Duration(j.RNG.Int63n(int64(interval)))
}
