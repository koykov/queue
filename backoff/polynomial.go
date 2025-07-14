package backoff

import (
	"math"
	"time"
)

// Polynomial grows interval by formula `value*attempt^k`
// where `n` - number of attempts and `k` - constant.
// Example (for 1 second interval and k=3):
// * 1st step - 1 second
// * 2nd step - 8 second
// * 3rd step - 27 second
// * ...
type Polynomial struct {
	K uint64
}

func (p Polynomial) Next(interval time.Duration, attempt int) time.Duration {
	return interval * time.Duration(math.Pow(float64(attempt), float64(p.K)))
}
