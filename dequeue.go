package blqueue

type DequeueWorker interface {
	Dequeue(x interface{}) error
}
