package rng

import (
	"math/rand"
	"sync"
	"time"
)

type Pool struct {
	p    sync.Pool
	New  func() rand.Source64
	once sync.Once
}

func (p *Pool) Get() RNG {
	p.once.Do(func() {
		if p.New == nil {
			p.New = func() rand.Source64 {
				s := rand.NewSource(time.Now().UnixNano())
				return any(s).(rand.Source64)
			}
		}
	})
	raw := p.p.Get()
	if raw == nil {
		r := rand.New(p.New())
		return r
	}
	return raw.(RNG)
}

func (p *Pool) Put(x RNG) {
	if x == nil {
		return
	}
	p.p.Put(x)
}
