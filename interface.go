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

type engine interface {
	init(config *Config) error
	put(itm *item, block bool) bool
	getc() chan item
	size() int
	cap() int
	close() error
}

// Worker describes queue worker interface.
type Worker interface {
	// Do process the item.
	Do(x any) error
}
