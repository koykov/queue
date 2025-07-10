package backoff

import (
	"math"
	"time"
)

// Exponential grows interval by formula `value*2^n`
// where `n` - number of attempts.
// Example (for 1 second interval):
// * 1st step - 1 second
// * 2nd step - 2 second
// * 3rd step - 4 second
// * 4th step - 8 second
// * ...
type Exponential struct{}

func (Exponential) Next(interval time.Duration, attempt int) time.Duration {
	return interval * time.Duration(math.Pow(2, float64(attempt)))
}
