package blqueue

// Dequeuer is and interface of dequeue component.
// Each queue must have an implementation of this component (see Config.Dequeuer).
type Dequeuer interface {
	// Dequeue processes an item taken from the queue.
	Dequeue(x interface{}) error
}
