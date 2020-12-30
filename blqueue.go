package queue

import "sync/atomic"

type Leaker interface {
	Catch(x interface{})
}

type BalancedLeakyQueue struct {
	BalancedQueue
	Leaker Leaker
}

func (q *BalancedLeakyQueue) Put(x interface{}) bool {
	if q.status == qstatusNil {
		q.once.Do(q.init)
	}

	if atomic.AddInt64(&q.spinlock, 1) >= spinlockLimit {
		q.rebalance()
	}
	select {
	case q.stream <- x:
		q.Metrics.QueuePut()
		atomic.AddInt64(&q.spinlock, -1)
		return true
	default:
		if q.Leaker != nil {
			q.Leaker.Catch(x)
		}
		q.Metrics.QueueLeak()
		return false
	}
}
