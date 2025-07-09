package backoff

import (
	"math"
	"time"
)

// Exponential grows interval by formula `value*2^n`
// where `n` - number of attempts.
type Exponential struct{}

func (Exponential) Next(interval time.Duration, attempt int) time.Duration {
	return interval * time.Duration(math.Pow(2, float64(attempt)))
}
