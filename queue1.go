package queue

import (
	"sync"
	"sync/atomic"
	"time"
)

type Status uint

const (
	StatusNil      Status = 0
	StatusFail            = 1
	StatusActive          = 2
	StatusThrottle        = 3
)

type flags struct {
	blocked, balanced bool
}

// todo remove old queue types and rename this to Queue
type Queue1 struct {
	flags  flags
	config Config

	status Status
	stream stream

	mux     sync.Mutex
	workers []*worker
	ctl     []ctl

	once sync.Once

	workersUp int32
	acqlock   uint32
	spinlock  int64

	Err error
}

func New(config Config) *Queue1 {
	q := &Queue1{
		config: config,
	}
	q.init()
	return q
}

func (q *Queue1) init() {
	c := &q.config

	if c.MetricsHandler == nil {
		c.MetricsHandler = &DummyMetrics{}
	}
	if c.Proc == nil {
		q.Err = ErrNoProc
		q.status = StatusFail
		return
	}

	if c.Workers > 0 && c.WorkersMin == 0 {
		c.WorkersMin = c.Workers
	}
	if c.Workers > 0 && c.WorkersMax == 0 {
		c.WorkersMax = c.Workers
	}

	if c.WakeupFactor <= 0 {
		c.WakeupFactor = defaultWakeupFactor
	}
	if c.SleepFactor <= 0 {
		c.SleepFactor = defaultSleepFactor
	}
	if c.WakeupFactor < c.SleepFactor {
		c.WakeupFactor = c.SleepFactor
	}

	q.flags.balanced = c.WorkersMin < c.WorkersMax
	q.flags.blocked = c.LeakyHandler == nil

	q.ctl = make([]ctl, c.WorkersMax)
	q.workers = make([]*worker, c.WorkersMax)
	var i uint32
	for i = 0; i < c.WorkersMax; i++ {
		c.MetricsHandler.WorkerSleep(i)
		q.ctl[i] = make(chan signal)
		q.workers[i] = &worker{
			idx:     i,
			status:  wstatusIdle,
			proc:    c.Proc,
			metrics: c.MetricsHandler,
		}
	}
	for i = 0; i < c.WorkersMin; i++ {
		go q.workers[i].observe(q.stream, q.ctl[i])
		q.ctl[i] <- signalInit
	}
	q.workersUp = int32(c.WorkersMin)

	if c.Heartbeat == 0 {
		c.Heartbeat = defaultHeartbeat
	}
	tickerHB := time.NewTicker(c.Heartbeat)
	go func() {
		for {
			select {
			case <-tickerHB.C:
				q.rebalance()
			}
		}
	}()

	q.status = StatusActive
}

func (q *Queue1) Enqueue(x interface{}) bool {
	if q.status == StatusNil {
		q.once.Do(q.init)
	}

	if atomic.AddInt64(&q.spinlock, 1) >= spinlockLimit {
		q.rebalance()
	}
	select {
	case q.stream <- x:
		q.config.MetricsHandler.QueuePut()
		atomic.AddInt64(&q.spinlock, -1)
		return true
	default:
		if q.config.LeakyHandler != nil {
			q.config.LeakyHandler.Catch(x)
		}
		q.config.MetricsHandler.QueueLeak()
		return false
	}
}

func (q *Queue1) String() string {
	return ""
}

func (q *Queue1) rebalance() {
	q.mux.Lock()
	defer q.mux.Unlock()
	if atomic.LoadUint32(&q.acqlock) == 1 {
		return
	}

	atomic.StoreUint32(&q.acqlock, 1)

	// Reset spinlock immediately to reduce amount of threads waiting for rebalance.
	q.spinlock = 0

	rate := q.lcRate()
	switch {
	case rate >= q.config.WakeupFactor:
		i := q.workersUp - 1
		go q.workers[i].observe(q.stream, q.ctl[i])
		q.ctl[i] <- signalResume
		atomic.AddInt32(&q.workersUp, 1)
	case rate <= q.config.SleepFactor:
		q.ctl[q.workersUp-1] <- signalSleep
		atomic.AddInt32(&q.workersUp, -1)
	case rate == 1:
		q.status = StatusThrottle
	default:
		q.status = StatusActive
	}

	atomic.StoreUint32(&q.acqlock, 0)
}

func (q *Queue1) lcRate() float32 {
	return float32(len(q.stream)) / float32(cap(q.stream))
}
