package blqueue

type DLQ interface {
	Enqueue(x interface{})
}
