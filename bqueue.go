package queue

import "sync"

const (
	defaultWakeupFactor = .75
	defaultSleepFactor  = .5
)

type BalancedQueue struct {
	queue

	once   sync.Once
	stream stream
	w      []*worker
	wc     uint32

	Size uint64

	proc Proc

	WakeupFactor float32
	SleepFactor  float32

	WorkersMin uint32
	WorkersMax uint32
}

func (q *BalancedQueue) Put(x interface{}) bool {
	q.rebalance()
	q.stream <- x
	return true
}

func (q *BalancedQueue) init() {
	q.stream = make(stream, q.Size)

	if q.WorkersMin <= 0 {
		q.WorkersMin = 1
	}
	if q.WorkersMax < q.WorkersMin {
		q.WorkersMax = q.WorkersMin
	}

	if q.WakeupFactor <= 0 {
		q.WakeupFactor = defaultWakeupFactor
	}
	if q.SleepFactor <= 0 {
		q.SleepFactor = defaultSleepFactor
	}
	if q.WakeupFactor < q.SleepFactor {
		q.WakeupFactor = q.SleepFactor
	}

	q.status = qstatusActive
}

func (q *BalancedQueue) rebalance() {
	if q.status == qstatusNil {
		q.once.Do(q.init)
	}

	rate := q.lcRate()
	switch {
	case rate >= q.WakeupFactor:
		// todo make new worker or wakeup sleeping worker and use it Observe() method in new goroutine.
	case rate <= q.SleepFactor:
		// todo sleep one of active workers.
	case rate == 1:
		q.status = qstatusThrottle
	default:
		q.status = qstatusActive
	}
}

func (q *BalancedQueue) lcRate() float32 {
	return float32(len(q.stream)) / float32(cap(q.stream))
}
