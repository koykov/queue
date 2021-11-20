package blqueue

type Dequeuer interface {
	Dequeue(x interface{}) error
}
