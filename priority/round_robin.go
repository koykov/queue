package priority

import (
	"sync/atomic"
)

// RoundRobin priority evaluator.
type RoundRobin struct {
	c uint64
}

func (p *RoundRobin) Eval(_ any) uint {
	return uint(atomic.AddUint64(&p.c, 1) % 100)
}
