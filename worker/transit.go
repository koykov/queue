package worker

import "github.com/koykov/blqueue"

// Transit represents worker that transits item to other queue.
type Transit struct {
	queue blqueue.Interface
}

// NewTransit makes transit worker with destination queue.
func NewTransit(queue blqueue.Interface) *Transit {
	w := Transit{queue: queue}
	return &w
}

func (w Transit) Do(x interface{}) error {
	if w.queue == nil {
		return blqueue.ErrNoQueue
	}
	return w.queue.Enqueue(x)
}

var _ = NewTransit
