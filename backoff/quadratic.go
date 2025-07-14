package backoff

import (
	"math"
	"time"
)

// Quadratic grows interval by formula `value*n^2`
// where `n` - number of attempts.
// Example (for 1 second interval):
// * 1st step - 1 second
// * 2nd step - 4 second
// * 3rd step - 9 second
// * ...
type Quadratic struct{}

func (Quadratic) Next(interval time.Duration, attempt int) time.Duration {
	return interval * time.Duration(math.Pow(float64(attempt), 2))
}
