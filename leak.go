package queue

// LeakDirection indicates the queue side to leak.
type LeakDirection uint

const (
	// LeakDirectionRear is a default direction that redirects to DLQ new incoming items.
	LeakDirectionRear LeakDirection = iota
	// LeakDirectionFront takes old item from queue front and redirects it to DLQ. Thus releases space for the new
	// incoming item in the queue.
	LeakDirectionFront

	defaultFrontLeakAttempts = 5
)
