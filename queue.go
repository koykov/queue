package queue

import (
	"encoding/json"
	"sync"
)

type qstatus uint
type wstatus uint
type signal uint

const (
	qstatusNil      qstatus = 0
	qstatusActive   qstatus = 1
	qstatusThrottle qstatus = 2

	wstatusIdle   wstatus = 0
	wstatusActive wstatus = 1
	wstatusSleep  wstatus = 2

	signalInit   signal = 0
	signalSleep  signal = 1
	signalResume signal = 2
	signalStop   signal = 3
)

type stream chan interface{}

type Queuer interface {
	Put(x interface{}) bool
	String() string
}

type Proc func(x interface{})

type Queue struct {
	status qstatus
	stream stream
	Size   uint64

	mux     sync.Mutex
	workers []*worker
	ctl     []ctl

	once sync.Once

	Key     string
	Proc    Proc
	Workers uint32
	Metrics MetricsWriter
}

func (q *Queue) init() {
	if q.Metrics == nil {
		q.Metrics = &DummyMetrics{}
	}
	if q.Proc == nil {
		q.Proc = DummyProc
	}

	q.stream = make(stream, q.Size)

	if q.Workers == 0 {
		q.Workers = 1
	}
	q.ctl = make([]ctl, q.Workers)
	q.workers = make([]*worker, q.Workers)
	var i uint32
	for i = 0; i < q.Workers; i++ {
		q.workers[i] = &worker{
			idx:     i,
			status:  wstatusIdle,
			proc:    q.Proc,
			metrics: q.Metrics,
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
	q.Metrics.QueuePut()
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
