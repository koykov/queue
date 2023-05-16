package queue

import "errors"

var (
	ErrNoConfig    = errors.New("no config provided")
	ErrNoCapacity  = errors.New("capacity must be greater than zero")
	ErrNoWorker    = errors.New("no worker provided")
	ErrNoWorkers   = errors.New("no workers available")
	ErrNoQueue     = errors.New("no queue provided")
	ErrQueueClosed = errors.New("queue closed")

	ErrNoQoS          = errors.New("no QoS config provided")
	ErrQoSUnknownAlgo = errors.New("unknown QoS scheduling algorithm")
	ErrQoSNoEvaluator = errors.New("no QoS priority evaluator provided")
	ErrQoSNoQueues    = errors.New("no QoS queues")

	ErrSchedMinGtMax = errors.New("min workers greater than max")
	ErrSchedZeroMax  = errors.New("max workers must be greater than 0")
	ErrSchedBadRange = errors.New("schedule range has bad format")
	ErrSchedBadTime  = errors.New("bad time provided")
	ErrSchedBadHour  = errors.New("hour outside range 0..23")
	ErrSchedBadMin   = errors.New("minute outside range 0..59")
	ErrSchedBadSec   = errors.New("second outside range 0..59")
	ErrSchedBadMsec  = errors.New("millisecond outside range 0..999")
)
