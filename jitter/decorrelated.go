package jitter

import (
	"math/rand"
	"sync"
	"time"
)

type dcjTuple struct {
	cntr int64
	rng  *rand.Rand
}

type Decorrelated struct {
	New      func() rand.Source64
	Min, Max time.Duration

	p    sync.Pool
	once sync.Once
}

func (j *Decorrelated) Apply(interval time.Duration) time.Duration {
	j.once.Do(func() {
		if j.New == nil {
			j.New = func() rand.Source64 {
				s := rand.NewSource(time.Now().UnixNano())
				return any(s).(rand.Source64)
			}
		}
	})
	raw := j.p.Get()
	if raw == nil {
		raw = &dcjTuple{
			cntr: int64(interval),
			rng:  rand.New(j.New()),
		}
	}
	defer j.p.Put(raw)

	t := raw.(*dcjTuple)
	cntr := t.rng.Int63n(3 * t.cntr)
	if cntr < int64(j.Min) {
		cntr = int64(j.Min)
	}
	if cntr > int64(j.Max) {
		cntr = int64(j.Max)
	}
	t.cntr = cntr
	return time.Duration(cntr)
}
