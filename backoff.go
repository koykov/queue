package queue

import "time"

type Backoff interface {
	Next(interval time.Duration, attempt int) time.Duration
}
