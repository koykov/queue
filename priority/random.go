package priority

import (
	"math/rand"
	"sync"

	"github.com/koykov/queue/rng"
)

// Random priority evaluator.
// Caution! Only for demo purposes.
type Random struct {
	RNG  rng.Interface
	once sync.Once
}

func (p *Random) Eval(_ any) uint {
	p.once.Do(func() {
		if p.RNG == nil {
			p.RNG = &rng.Pool{}
		}
	})
	return uint(rand.Intn(100)) + 1
}
