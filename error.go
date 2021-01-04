package queue

import "errors"

var (
	ErrNoSize   = errors.New("size size must be greater than zero")
	ErrNoProc   = errors.New("no processing function provided")
	ErrNoWorker = errors.New("no workers available")
)
