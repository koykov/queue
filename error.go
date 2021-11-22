package blqueue

import "errors"

var (
	ErrNoKey      = errors.New("queue must have name")
	ErrNoSize     = errors.New("size must be greater than zero")
	ErrNoDequeuer = errors.New("no dequeuer component provided")
	ErrNoWorker   = errors.New("no workers available")

	ErrSchedMinGtMax = errors.New("min workers greater than max")
	ErrSchedZeroMax  = errors.New("max workers must be greater than 0")
	ErrSchedBadRange = errors.New("schedule range has bad format")
	ErrSchedBadTime  = errors.New("bad time provided")
	ErrSchedBadHour  = errors.New("hour outside range 0..23")
	ErrSchedBadMin   = errors.New("minute outside range 0..59")
	ErrSchedBadSec   = errors.New("second outside range 0..59")
	ErrSchedBadMsec  = errors.New("millisecond outside range 0..999")
)
