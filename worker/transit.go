package worker

import "github.com/koykov/queue"

// Transit represents worker that transits item to other queue.
type Transit struct {
	queue queue.Interface
}

// TransitTo makes transit worker with destination queue.
func TransitTo(queue queue.Interface) *Transit {
	w := Transit{queue: queue}
	return &w
}

func (w Transit) Do(x interface{}) error {
	if w.queue == nil {
		return queue.ErrNoQueue
	}
	return w.queue.Enqueue(x)
}

var _ = TransitTo
