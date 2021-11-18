package blqueue

import (
	"encoding/json"
	"math"
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

	spinlockLimit = 1000

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
	spinlock  uint32
	enqlock   uint32

	Err error
}

func New(config *Config) *Queue {
	q := &Queue{
		config: config.Copy(),
	}
	q.once.Do(q.init)
	return q
}

func (q *Queue) init() {
	c := q.config

	if c.Size == 0 {
		q.Err = ErrNoSize
		q.status = StatusFail
		return
	}
	if c.DequeueWorker == nil {
		q.Err = ErrNoDequeue
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
		q.workers[i].signal(sigInit)
		go q.workers[i].dequeue(q.stream)
	}
	q.workersUp = int32(c.WorkersMin)

	if c.Heartbeat == 0 {
		c.Heartbeat = defaultHeartbeat
	}
	if q.CheckBit(flagBalanced) {
		tickerHB := time.NewTicker(c.Heartbeat)
		go func() {
			for {
				select {
				case <-tickerHB.C:
					q.rebalance(false)
					if q.lcRate() == 0 && q.getStatus() == StatusClose {
						return
					}
				}
			}
		}()
	}

	q.setStatus(StatusActive)
}

func (q *Queue) Enqueue(x interface{}) bool {
	q.once.Do(q.init)
	if q.getStatus() == StatusClose {
		return false
	}

	atomic.AddUint32(&q.enqlock, 1)
	defer atomic.AddUint32(&q.enqlock, math.MaxUint32)

	if q.CheckBit(flagBalanced) {
		if atomic.AddUint32(&q.spinlock, 1) >= spinlockLimit {
			q.rebalance(true)
		}
	}
	q.m().QueuePut()
	if q.CheckBit(flagLeaky) {
		select {
		case q.stream <- x:
			atomic.AddUint32(&q.spinlock, math.MaxUint32)
			return true
		default:
			q.c().DLQ.Enqueue(x)
			q.m().QueueLeak()
			return false
		}
	} else {
		q.stream <- x
		atomic.AddUint32(&q.spinlock, math.MaxUint32)
		return true
	}
}

func (q *Queue) Close() {
	q.setStatus(StatusClose)
	for atomic.LoadUint32(&q.enqlock) > 0 {
	}
	close(q.stream)
}

func (q *Queue) ForceClose() {
	// todo implement me
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
	atomic.StoreUint32(&q.spinlock, 0)

	rate := q.lcRate()
	if q.c().Verbose(VerboseInfo) {
		msg := "rebalance: rate %f, workers %d"
		if force {
			msg = "force rebalance: rate %f, workers %d"
		}
		q.l().Printf(msg, rate, atomic.LoadInt32(&q.workersUp))
	}

	q.checkAsleep()

	switch {
	case rate == 0 && q.getStatus() == StatusClose:
		for i := 0; uint32(i) < q.c().WorkersMax; i++ {
			if q.workers[i].getStatus() == WorkerStatusActive || q.workers[i].getStatus() == WorkerStatusSleep {
				q.workers[i].signal(sigForceStop)
			}
		}
	case rate >= q.c().WakeupFactor:
		if uint32(q.getWorkersUp()) == q.c().WorkersMax {
			return
		}
		for i := q.c().WorkersMin; i < q.c().WorkersMax; i++ {
			ws := q.workers[i].getStatus()
			if ws == WorkerStatusActive {
				continue
			}
			if q.workers[i].getStatus() == WorkerStatusIdle {
				q.workers[i].signal(sigInit)
				go q.workers[i].dequeue(q.stream)
			} else {
				q.workers[i].signal(sigWakeup)
			}
			atomic.AddInt32(&q.workersUp, 1)
			break
		}
	case rate <= q.c().SleepFactor:
		if uint32(q.getWorkersUp()) == q.c().WorkersMin {
			return
		}
		for i := q.c().WorkersMax - 1; i >= q.c().WorkersMin; i-- {
			if q.workers[i].getStatus() == WorkerStatusActive {
				q.workers[i].signal(sigSleep)
				atomic.AddInt32(&q.workersUp, -1)
				break
			}
		}
	case rate == 1:
		q.setStatus(StatusThrottle)
	default:
		if q.getStatus() == StatusThrottle {
			q.setStatus(StatusActive)
		}
	}
}

func (q *Queue) checkAsleep() {
	for i := q.c().WorkersMax - 1; i >= q.c().WorkersMin; i-- {
		if q.workers[i].getStatus() == WorkerStatusSleep && q.workers[i].sleptEnough() {
			q.workers[i].signal(sigStop)
		}
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
