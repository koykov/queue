package blqueue

// DLQ is an interface of dead letter queues interface.
// When queue is full but new incoming items continue to flow, all leaked items puts to the DLQ and may be processed.
type DLQ interface {
	// Enqueue takes a leaked item and process it.
	Enqueue(x interface{})
}
