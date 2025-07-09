package backoff

import "time"

// Linear multiplies interval to attempts value.
type Linear struct{}

func (Linear) Next(interval time.Duration, attempt int) time.Duration {
	return interval * time.Duration(attempt)
}
