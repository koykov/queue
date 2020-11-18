package queue

const (
	defaultWorkersLimit  = 8
	newWorkerThreshold   = .75
	sleepWorkerThreshold = .5
)

type BalancedQueue struct {
	stream Stream

	Worker                        Worker
	WorkersInit, WorkersLimit, wc uint32
}

func (q *BalancedQueue) Put(x interface{}) bool {
	// todo init queue once
	q.rebalance()
	q.stream <- x
	return true
}

func (q *BalancedQueue) rebalance() {
	rate := q.lcRate()
	if rate >= newWorkerThreshold {
		// todo make new worker or wakeup sleeping worker and use it Observe() method in new goroutine.
	} else if rate <= sleepWorkerThreshold {
		// todo sleep one of active workers.
	}
}

func (q *BalancedQueue) lcRate() float32 {
	return float32(len(q.stream)) / float32(cap(q.stream))
}
