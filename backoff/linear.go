package backoff

import "time"

// Linear multiplies interval to attempts value.
// Example (for 1 second interval):
// * 1st step - 1 second
// * 2nd step - 2 second
// * 3rd step - 3 second
// * ...
type Linear struct{}

func (Linear) Next(interval time.Duration, attempt int) time.Duration {
	return interval * time.Duration(attempt)
}
