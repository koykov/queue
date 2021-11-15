package blqueue

import "errors"

var (
	ErrNoSize    = errors.New("size must be greater than zero")
	ErrNoDequeue = errors.New("no dequeue workers provided")
	ErrNoWorker  = errors.New("no workers available")
)
