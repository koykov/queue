package queue

import "time"

type Jitter interface {
	Apply(interval time.Duration) time.Duration
}
