package rng

// RNG describes random number generator.
type RNG interface {
	Int63n(n int64) int64
	Intn(n int) int
}
