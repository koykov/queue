package queue

import (
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Status uint

const (
	StatusNil Status = iota
	StatusFail
	StatusActive
	StatusThrottle

	spinlockLimit = 1000
)

type flags struct {
	balanced, leaky bool
}

type stream chan interface{}

type Queue struct {
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

type Proc func(interface{})

type Leaker interface {
	Catch(x interface{})
}

func New(config Config) *Queue {
	q := &Queue{
		config: config,
	}
	q.init()
	return q
}

func (q *Queue) init() {
	c := &q.config

	if c.Size == 0 {
		q.Err = ErrNoSize
		q.status = StatusFail
		return
	}
	if c.Proc == nil {
		q.Err = ErrNoProc
		q.status = StatusFail
		return
	}

	if c.MetricsHandler == nil {
		c.MetricsHandler = &DummyMetrics{}
	}

	if c.Workers > 0 && c.WorkersMin == 0 {
		c.WorkersMin = c.Workers
	}
	if c.Workers > 0 && c.WorkersMax == 0 {
		c.WorkersMax = c.Workers
	}
	if c.WorkersMax == 0 {
		q.Err = ErrNoWorker
		q.status = StatusFail
		return
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
	if c.SleepTimeout == 0 {
		c.SleepTimeout = defaultSleepTimeout
	}

	q.stream = make(stream, c.Size)

	q.flags.balanced = c.WorkersMin < c.WorkersMax
	q.flags.leaky = c.LeakyHandler != nil

	q.ctl = make([]ctl, c.WorkersMax)
	q.workers = make([]*worker, c.WorkersMax)
	var i uint32
	for i = 0; i < c.WorkersMax; i++ {
		c.MetricsHandler.WorkerSleep(i)
		q.ctl[i] = make(chan signal, 1)
		q.workers[i] = &worker{
			idx:     i,
			status:  wstatusIdle,
			proc:    c.Proc,
			metrics: c.MetricsHandler,
		}
	}
	c.MetricsHandler.WorkerSetup(0, 0, uint(c.WorkersMax))

	for i = 0; i < c.WorkersMin; i++ {
		go q.workers[i].observe(q.stream, q.ctl[i])
		q.ctl[i] <- signalInit
	}
	q.workersUp = int32(c.WorkersMin)

	if c.Heartbeat == 0 {
		c.Heartbeat = defaultHeartbeat
	}
	if q.flags.balanced {
		tickerHB := time.NewTicker(c.Heartbeat)
		go func() {
			for {
				select {
				case <-tickerHB.C:
					q.rebalance()
				}
			}
		}()
	}

	q.status = StatusActive
}

func (q *Queue) Enqueue(x interface{}) bool {
	if q.status == StatusNil {
		q.once.Do(q.init)
	}

	if q.flags.balanced {
		if atomic.AddInt64(&q.spinlock, 1) >= spinlockLimit {
			q.rebalance()
		}
	}
	q.config.MetricsHandler.QueuePut()
	if q.flags.leaky {
		select {
		case q.stream <- x:
			atomic.AddInt64(&q.spinlock, -1)
			return true
		default:
			q.config.LeakyHandler.Catch(x)
			q.config.MetricsHandler.QueueLeak()
			return false
		}
	} else {
		q.stream <- x
		atomic.AddInt64(&q.spinlock, -1)
		return true
	}
}

func (q *Queue) rebalance() {
	q.mux.Lock()
	defer func() {
		atomic.StoreUint32(&q.acqlock, 0)
		q.mux.Unlock()
	}()
	if atomic.LoadUint32(&q.acqlock) == 1 {
		return
	}

	atomic.StoreUint32(&q.acqlock, 1)

	// Reset spinlock immediately to reduce amount of threads waiting for rebalance.
	q.spinlock = 0

	rate := q.lcRate()
	log.Println("rate", rate)
	switch {
	case rate >= q.config.WakeupFactor:
		i := q.workersUp
		if uint32(i) == q.config.WorkersMax {
			return
		}
		if q.workers[i].status == wstatusIdle {
			go q.workers[i].observe(q.stream, q.ctl[i])
			q.ctl[i] <- signalInit
		} else {
			q.ctl[i] <- signalResume
		}
		atomic.AddInt32(&q.workersUp, 1)
	case rate <= q.config.SleepFactor:
		i := q.workersUp - 1
		if uint32(i) < q.config.WorkersMin {
			return
		}
		atomic.AddInt32(&q.workersUp, -1)
		q.ctl[i] <- signalSleep

		for i := q.workersUp; uint32(i) < q.config.WorkersMax; i++ {
			if q.workers[i].status == wstatusSleep && q.workers[i].lastTS.Add(q.config.SleepTimeout).Before(time.Now()) {
				q.ctl[i] <- signalStop
			}
		}
	case rate == 1:
		q.status = StatusThrottle
	default:
		q.status = StatusActive
	}
}

func (q *Queue) lcRate() float32 {
	return float32(len(q.stream)) / float32(cap(q.stream))
}

func (q *Queue) String() string {
	var out = struct {
		Size          uint64        `json:"size"`
		Workers       uint32        `json:"workers"`
		Heartbeat     time.Duration `json:"heartbeat"`
		WorkersMin    uint32        `json:"workers_min"`
		WorkersMax    uint32        `json:"workers_max"`
		WakeupFactor  float32       `json:"wakeup_factor"`
		SleepFactor   float32       `json:"sleep_factor"`
		Status        string        `json:"status"`
		FullnessRate  float32       `json:"fullness_rate"`
		WorkersIdle   int           `json:"workers_idle"`
		WorkersActive int           `json:"workers_active"`
		WorkersSleep  int           `json:"workers_sleep"`
	}{}

	out.Size = q.config.Size
	out.Workers = q.config.Workers
	out.Heartbeat = q.config.Heartbeat
	out.WorkersMin = q.config.WorkersMin
	out.WorkersMax = q.config.WorkersMax
	out.WakeupFactor = q.config.WakeupFactor
	out.SleepFactor = q.config.SleepFactor

	switch q.status {
	case StatusNil:
		out.Status = "inactive"
	case StatusFail:
		out.Status = "fail"
	case StatusActive:
		out.Status = "active"
	case StatusThrottle:
		out.Status = "throttle"
	}
	out.FullnessRate = q.lcRate()

	for _, w := range q.workers {
		if w == nil {
			out.WorkersIdle++
		} else {
			switch w.status {
			case wstatusIdle:
				out.WorkersIdle++
			case wstatusActive:
				out.WorkersActive++
			case w.status:
				out.WorkersSleep++
			}
		}
	}

	b, _ := json.Marshal(out)

	return string(b)
}
