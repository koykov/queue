package blqueue

import "errors"

var (
	ErrNoKey      = errors.New("queue must have name")
	ErrNoSize     = errors.New("size must be greater than zero")
	ErrNoDequeuer = errors.New("no dequeuer component provided")
	ErrNoWorker   = errors.New("no workers available")

	ErrSchedBadRange = errors.New("schedule range has bad format")
)
