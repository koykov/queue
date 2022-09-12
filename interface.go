package blqueue

// Interface describes queue interface.
type Interface interface {
	// Enqueue puts item to the queue.
	Enqueue(x interface{}) error
	// Size return actual size of the queue.
	Size() int
	// Capacity return max size of the queue.
	Capacity() int
	// Rate returns size to capacity ratio.
	Rate() float32
	// Close gracefully stops the queue.
	Close() error
}

// Worker describes queue worker interface.
type Worker interface {
	// Do process the item.
	Do(x interface{}) error
}
