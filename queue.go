package queue

type Queuer interface {
	Put(x interface{}) bool
}

type Stream chan interface{}

type Worker interface {
	Observe(stream Stream)
	Sleep()
	Wakeup()
}
