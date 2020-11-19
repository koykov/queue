package queue

import "sync"

const (
	defaultWakeupFactor = .75
	defaultSleepFactor  = .5
)

type BalancedQueue struct {
	once   sync.Once
	stream Stream
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
	q.stream = make(Stream, q.Size)

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
}

func (q *BalancedQueue) rebalance() {
	if q.stream == nil {
		q.once.Do(q.init)
	}

	rate := q.lcRate()
	if rate >= q.WakeupFactor {
		// todo make new worker or wakeup sleeping worker and use it Observe() method in new goroutine.
	} else if rate <= q.SleepFactor {
		// todo sleep one of active workers.
	}
}

func (q *BalancedQueue) lcRate() float32 {
	return float32(len(q.stream)) / float32(cap(q.stream))
}
