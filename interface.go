package queue

// Enqueuer describes component that can enqueue items.
type Enqueuer interface {
	// Enqueue puts item to the queue.
	Enqueue(x any) error
}

// Interface describes queue interface.
type Interface interface {
	Enqueuer
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
	Do(x any) error
}

// PriorityEvaluator calculates priority percent of items comes to PQ.
type PriorityEvaluator interface {
	// Eval returns priority percent of x.
	Eval(x any) uint
}

// Internal engine definition.
type engine interface {
	// Init engine using config.
	init(config *Config) error
	// Put new item to the engine in blocking or non-blocking mode.
	// Returns true/false for non-blocking mode.
	// Always returns true in blocking mode.
	enqueue(itm *item, block bool) bool
	// Get item from the engine in blocking or non-blocking mode.
	// Returns true/false for non-blocking mode.
	// Always returns true in blocking mode.
	dequeue(block bool) (item, bool)
	// Return count of collected items.
	size() int
	// Returns the whole capacity.
	cap() int
	// Close the engine.
	close(force bool) error
}
