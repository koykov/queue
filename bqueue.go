package queue

type BalancedQueue struct {
	stream Stream
}

func (q *BalancedQueue) Put(x interface{}) bool {
	q.stream <- x
	return true
}
