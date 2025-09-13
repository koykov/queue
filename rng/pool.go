package rng

import (
	"math/rand"
	"sync"
	"time"
)

// Pool provides thread-safe container of RNG instances.
// Using New field you may specify your own source for RNG.
// In general Pool implements typical RNG methods, but in addition it has Get and Put methods to acquire/release
// rand.Rand objects. This need in order to deny to pass simple thread-unsafe rand.Rand instance instead of pool.
type Pool struct {
	p    sync.Pool
	New  func() rand.Source64
	once sync.Once
}

func (r *Pool) Int() int {
	rng := r.Get()
	defer r.Put(rng)
	return rng.Int()
}

func (r *Pool) Intn(n int) int {
	rng := r.Get()
	defer r.Put(rng)
	return rng.Intn(n)
}

func (r *Pool) Int31() int32 {
	rng := r.Get()
	defer r.Put(rng)
	return rng.Int31()
}

func (r *Pool) Int31n(n int32) int32 {
	rng := r.Get()
	defer r.Put(rng)
	return rng.Int31n(n)
}

func (r *Pool) Int63() int64 {
	rng := r.Get()
	defer r.Put(rng)
	return rng.Int63()
}

func (r *Pool) Int63n(n int64) int64 {
	rng := r.Get()
	defer r.Put(rng)
	return rng.Int63n(n)
}

func (r *Pool) Uint32() uint32 {
	rng := r.Get()
	defer r.Put(rng)
	return rng.Uint32()
}

func (r *Pool) Uint64() uint64 {
	rng := r.Get()
	defer r.Put(rng)
	return rng.Uint64()
}

func (r *Pool) Float32() float32 {
	rng := r.Get()
	defer r.Put(rng)
	return rng.Float32()
}

func (r *Pool) Float64() float64 {
	rng := r.Get()
	defer r.Put(rng)
	return rng.Float64()
}

func (r *Pool) Get() *rand.Rand {
	r.once.Do(func() {
		if r.New == nil {
			r.New = func() rand.Source64 {
				s := rand.NewSource(time.Now().UnixNano())
				return any(s).(rand.Source64)
			}
		}
	})
	raw := r.p.Get()
	if raw == nil {
		rng := rand.New(r.New())
		return rng
	}
	return raw.(*rand.Rand)
}

func (r *Pool) Put(x *rand.Rand) {
	if x == nil {
		return
	}
	r.p.Put(x)
}
