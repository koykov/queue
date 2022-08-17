package blqueue

// Interface describes queue interface.
type Interface interface {
	// Enqueue puts item to the queue.
	Enqueue(x interface{}) error
	// Rate returns a fullness rate of the queue.
	Rate() float32
	// Close gracefully stops the queue.
	Close() error
}
