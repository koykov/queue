package backoff

import (
	"math"
	"time"
)

// Quadratic grows interval by formula `value*n^2`
// where `n` - number of attempts.
type Quadratic struct{}

func (Quadratic) Next(interval time.Duration, attempt int) time.Duration {
	return interval * time.Duration(math.Pow(float64(attempt), 2))
}
