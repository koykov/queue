package queue

type signal uint
type status uint

const (
	signalInit   signal = 0
	signalSleep         = 1
	signalResume        = 2
	signalStop          = 3

	statusActive status = 0
	statusIdle          = 1
	statusSleep         = 2
)

type Queuer interface {
	Put(x interface{}) bool
}

type Stream chan interface{}
type Proc func(x interface{})

type worker struct {
	status status
	proc   Proc
}
