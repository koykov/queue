package priority

import "math/rand"

// Random priority evaluator.
// Caution! Only for demo purposes.
type Random struct{}

func (Random) Eval(_ any) uint {
	return uint(rand.Intn(100)) + 1
}
