package queue

type BalancedLeakyQueue struct {
	stream Stream
}

func (q *BalancedLeakyQueue) Put(x interface{}) bool {
	select {
	case q.stream <- x:
		return true
	default:
		return false
	}
}
