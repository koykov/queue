package jitter

import (
	"math/rand"
	"sync"
	"time"
)

type base struct {
	r    *rand.Rand
	once sync.Once
}

func (b *base) init() {
	b.once.Do(func() {
		b.r = rand.New(rand.NewSource(time.Now().UnixNano()))
	})
}
