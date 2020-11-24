package queue

import (
	"sync"
)

type status uint
type signal uint

const (
	qstatusNil      status = 0
	qstatusActive          = 1
	qstatusThrottle        = 2

	wstatusIdle   status = 0
	wstatusActive        = 1
	wstatusSleep         = 2

	signalInit   signal = 0
	signalSleep         = 1
	signalResume        = 2
	signalStop          = 3
)

type stream chan interface{}

type Queuer interface {
	Put(x interface{}) bool
}

type Proc func(x interface{})

type Queue struct {
	status status
	stream stream
	Size   uint64

	mux     sync.Mutex
	workers []*worker
	ctl     []ctl

	proc Proc

	once sync.Once

	Workers uint32
}

func (q *Queue) init() {
	q.stream = make(stream, q.Size)

	if q.Workers == 0 {
		q.Workers = 1
	}
	q.workers = make([]*worker, q.Workers)
	var i uint32
	for i = 0; i < q.Workers; i++ {
		q.workers[i] = &worker{
			status: wstatusIdle,
			proc:   q.proc,
		}
		q.ctl[i] = make(ctl)

		go q.workers[i].observe(q.stream, q.ctl[i])
		q.ctl[i] <- signalInit
	}

	q.status = qstatusActive
}

func (q *Queue) Put(x interface{}) bool {
	if q.status == qstatusNil {
		q.once.Do(q.init)
	}

	q.stream <- x
	return true
}
