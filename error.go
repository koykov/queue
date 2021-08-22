package blqueue

import "errors"

var (
	ErrNoSize     = errors.New("size must be greater than zero")
	ErrNoDequeuer = errors.New("no dequeue handler provided")
	ErrNoWorker   = errors.New("no workers available")
)
