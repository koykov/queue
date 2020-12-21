package queue

import (
	"encoding/json"
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
	String() string
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

	Key     string
	Workers uint32
	Metrics MetricsWriter
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

func (q *Queue) String() string {
	var out = &struct {
		Key           string `json:"key"`
		Status        string `json:"status"`
		Size          uint64 `json:"size"`
		WorkersActive int    `json:"workers_active"`
	}{}
	out.Key = q.Key
	out.Size = q.Size

	switch q.status {
	case qstatusNil:
		out.Status = "inactive"
	case qstatusActive:
		out.Status = "active"
	case qstatusThrottle:
		out.Status = "throttle"
	}

	out.WorkersActive = int(q.Workers)

	b, _ := json.Marshal(out)

	return string(b)
}
