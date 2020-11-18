package queue

type Leaker interface {
	Catch(x interface{})
}

type BalancedLeakyQueue struct {
	BalancedQueue
	Leaker Leaker
}

func (q *BalancedLeakyQueue) Put(x interface{}) bool {
	q.rebalance()
	select {
	case q.stream <- x:
		return true
	default:
		if q.Leaker != nil {
			q.Leaker.Catch(x)
		}
		return false
	}
}
