package queue

import "errors"

var (
	ErrNoProc = errors.New("no processing function provided")
)
