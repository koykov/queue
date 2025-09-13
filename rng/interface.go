package rng

import "math/rand"

// Interface describes random number generator.
type Interface interface {
	Get() *rand.Rand
	Put(x *rand.Rand)

	Int() int
	Intn(n int) int

	Int31() int32
	Int31n(n int32) int32

	Int63() int64
	Int63n(n int64) int64

	Uint32() uint32
	Uint64() uint64

	Float32() float32
	Float64() float64
}
