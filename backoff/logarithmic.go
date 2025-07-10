package backoff

import (
	"math"
	"time"
)

// Logarithmic grows interval by formula `value*log(attempt+1)`
// where `n` - number of attempts.
// Example (for 10 second interval and k=3):
// * 1st step - 7 second
// * 2nd step - 11 second
// * 3rd step - 14 second
// * ...
type Logarithmic struct{}

func (Logarithmic) Next(interval time.Duration, attempt int) time.Duration {
	return interval * time.Duration(math.Log(float64(attempt)+1))
}
