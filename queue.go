package blqueue

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/koykov/bitset"
)

type Status uint32

const (
	StatusNil Status = iota
	StatusFail
	StatusActive
	StatusThrottle
	StatusClose

	spinlockLimit  = 1000
	idleCountLimit = 10

	flagBalanced = 0
	flagLeaky    = 1
)

type stream chan interface{}

type Queue struct {
	bitset.Bitset
	config *Config

	status Status
	stream stream

	mux     sync.Mutex
	workers []*worker

	once sync.Once

	workersUp int32
	acqlock   uint32
	spinlock  int64
	enqlock   int64

	Err error
}

func New(config *Config) *Queue {
	q := &Queue{
		config: config.Copy(),
	}
	q.init()
	return q
}

func (q *Queue) init() {
	c := q.config

	if c.Size == 0 {
		q.Err = ErrNoSize
		q.status = StatusFail
		return
	}
	if c.Dequeuer == nil {
		q.Err = ErrNoDequeuer
		q.status = StatusFail
		return
	}

	if c.MetricsWriter == nil {
		c.MetricsWriter = &DummyMetrics{}
	}

	if c.Logger == nil {
		c.Logger = &DummyLog{}
	}
	if c.VerbosityLevel == 0 {
		c.VerbosityLevel = VerboseNone
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

	q.SetBit(flagBalanced, c.WorkersMin < c.WorkersMax)
	q.SetBit(flagLeaky, c.DLQ != nil)

	q.workers = make([]*worker, c.WorkersMax)
	var i uint32
	for i = 0; i < c.WorkersMax; i++ {
		q.m().WorkerSleep(i)
		q.workers[i] = makeWorker(i, c)
	}
	q.m().WorkerSetup(0, 0, uint(c.WorkersMax))

	for i = 0; i < c.WorkersMin; i++ {
		go q.workers[i].dequeue(q.stream)
		q.workers[i].init()
	}
	q.workersUp = int32(c.WorkersMin)

	if c.Heartbeat == 0 {
		c.Heartbeat = defaultHeartbeat
	}
	if q.CheckBit(flagBalanced) {
		tickerHB := time.NewTicker(c.Heartbeat)
		go func() {
			idleC := 0
			for {
				select {
				case <-tickerHB.C:
					q.rebalance(false)
					if q.lcRate() == 0 && q.getStatus() == StatusClose {
						idleC++
					} else {
						idleC = 0
					}
					if idleC > idleCountLimit {
						return
					}
				}
			}
		}()
	}

	q.setStatus(StatusActive)
}

func (q *Queue) Enqueue(x interface{}) bool {
	if q.getStatus() == StatusNil {
		q.once.Do(q.init)
	}
	if q.getStatus() == StatusClose {
		return false
	}

	atomic.AddInt64(&q.enqlock, 1)
	defer atomic.AddInt64(&q.enqlock, -1)

	if q.CheckBit(flagBalanced) {
		if atomic.AddInt64(&q.spinlock, 1) >= spinlockLimit {
			q.rebalance(true)
		}
	}
	q.m().QueuePut()
	if q.CheckBit(flagLeaky) {
		select {
		case q.stream <- x:
			atomic.AddInt64(&q.spinlock, -1)
			return true
		default:
			q.c().DLQ.Enqueue(x)
			q.m().QueueLeak()
			return false
		}
	} else {
		q.stream <- x
		atomic.AddInt64(&q.spinlock, -1)
		return true
	}
}

func (q *Queue) Close() {
	q.setStatus(StatusClose)
	for atomic.LoadInt64(&q.enqlock) > 0 {
	}
	close(q.stream)
}

func (q *Queue) rebalance(force bool) {
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
	atomic.StoreInt64(&q.spinlock, 0)

	rate := q.lcRate()
	if force && q.c().Verbose(VerboseWarn) {
		q.l().Printf("force rebalance on rate %f", rate)
	} else if q.c().Verbose(VerboseInfo) {
		q.l().Printf("rebalance on rate %f", rate)
	}
	switch {
	case rate == 0 && q.getStatus() == StatusClose:
		for i := 0; uint32(i) < q.c().WorkersMax; i++ {
			if q.workers[i].getStatus() == WorkerStatusActive || q.workers[i].getStatus() == WorkerStatusSleep {
				q.workers[i].stop(true)
			}
		}
	case rate >= q.c().WakeupFactor:
		i := q.workersUp
		if uint32(i) == q.c().WorkersMax {
			return
		}
		if q.workers[i].getStatus() == WorkerStatusIdle {
			go q.workers[i].dequeue(q.stream)
			q.workers[i].init()
		} else {
			q.workers[i].wakeup()
		}
		atomic.AddInt32(&q.workersUp, 1)
	case rate <= q.c().SleepFactor:
		i := q.getWorkersUp() - 1
		if (i < int32(q.c().WorkersMin) && q.getStatus() != StatusClose) || i < 0 {
			return
		}
		wu := atomic.AddInt32(&q.workersUp, -1)
		q.workers[i].sleep()

		for i := wu; uint32(i) < q.c().WorkersMax; i++ {
			if q.workers[i].getStatus() == WorkerStatusSleep && q.workers[i].lastTS.Add(q.c().SleepTimeout).Before(time.Now()) {
				q.workers[i].stop(false)
			}
		}
	case rate == 1:
		q.setStatus(StatusThrottle)
	default:
		q.setStatus(StatusActive)
	}
}

func (q *Queue) lcRate() float32 {
	return float32(len(q.stream)) / float32(cap(q.stream))
}

func (q *Queue) getWorkersUp() int32 {
	return atomic.LoadInt32(&q.workersUp)
}

func (q *Queue) setStatus(status Status) {
	atomic.StoreUint32((*uint32)(&q.status), uint32(status))
}

func (q *Queue) getStatus() Status {
	return Status(atomic.LoadUint32((*uint32)(&q.status)))
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
	case StatusClose:
		out.Status = "close"
	}
	out.FullnessRate = q.lcRate()

	for _, w := range q.workers {
		if w == nil {
			out.WorkersIdle++
		} else {
			switch w.getStatus() {
			case WorkerStatusIdle:
				out.WorkersIdle++
			case WorkerStatusActive:
				out.WorkersActive++
			default:
				out.WorkersSleep++
			}
		}
	}

	b, _ := json.Marshal(out)

	return string(b)
}

func (q *Queue) c() *Config {
	return q.config
}

func (q *Queue) m() MetricsWriter {
	return q.config.MetricsWriter
}

func (q *Queue) l() Logger {
	return q.config.Logger
}
