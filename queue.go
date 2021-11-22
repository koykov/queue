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

	flagBalanced = 0
	flagLeaky    = 1
)

type stream chan interface{}

type Queue struct {
	bitset.Bitset
	config  *Config
	schedID int
	wmax    uint32

	status Status
	stream stream

	mux     sync.Mutex
	workers []*worker

	once sync.Once

	workersUp int32
	c9nlock   uint32
	spinlock  int64
	enqlock   int64

	Err error
}

func New(config *Config) (*Queue, error) {
	q := &Queue{
		config: config.Copy(),
	}
	q.once.Do(q.init)
	return q, q.Err
}

func (q *Queue) init() {
	c := q.config

	if len(c.Key) == 0 {
		q.Err = ErrNoKey
		q.status = StatusFail
		return
	}
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

	if c.ForceCalibrationLimit == 0 {
		c.ForceCalibrationLimit = defaultForceCalibrationLimit
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

	q.SetBit(flagBalanced, c.WorkersMin < c.WorkersMax || c.Schedule != nil)
	q.SetBit(flagLeaky, c.DLQ != nil)

	q.wmax = q.rtWorkersMax()
	var workersMin, workersMax uint32
	workersMin, workersMax, _, _, q.schedID = q.rtParams()

	q.workers = make([]*worker, q.wmax)
	var i uint32
	for i = 0; i < q.wmax; i++ {
		q.m().WorkerSleep(q.k(), i)
		q.workers[i] = makeWorker(i, c)
	}
	q.m().WorkerSetup(q.k(), 0, 0, uint(workersMax))

	for i = 0; i < workersMin; i++ {
		q.workers[i].signal(sigInit)
		go q.workers[i].dequeue(q.stream)
	}
	q.workersUp = int32(workersMin)

	if c.Heartbeat == 0 {
		c.Heartbeat = defaultHeartbeat
	}
	if q.CheckBit(flagBalanced) {
		tickerHB := time.NewTicker(c.Heartbeat)
		go func() {
			for {
				select {
				case <-tickerHB.C:
					q.calibrate(false)
					if q.Rate() == 0 && q.getStatus() == StatusClose {
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

	atomic.AddInt64(&q.enqlock, 1)
	defer atomic.AddInt64(&q.enqlock, -1)

	if q.CheckBit(flagBalanced) {
		defer atomic.AddInt64(&q.spinlock, -1)
		if atomic.AddInt64(&q.spinlock, 1) >= int64(q.c().ForceCalibrationLimit) {
			q.calibrate(true)
		}
	}
	q.m().QueuePut(q.k())
	if q.CheckBit(flagLeaky) {
		select {
		case q.stream <- x:
			return true
		default:
			q.c().DLQ.Enqueue(x)
			q.m().QueueLeak(q.k())
			return false
		}
	} else {
		q.stream <- x
		return true
	}
}

func (q *Queue) Rate() float32 {
	return float32(len(q.stream)) / float32(cap(q.stream))
}

func (q *Queue) Close() {
	if q.l() != nil {
		q.l().Printf("queue #%s caught close signal", q.k())
	}
	q.setStatus(StatusClose)
	for atomic.LoadInt64(&q.enqlock) > 0 {
	}
	close(q.stream)
}

func (q *Queue) ForceClose() {
	if q.l() != nil {
		q.l().Printf("queue #%s caught force close signal", q.k())
	}
	// todo implement me
}

func (q *Queue) calibrate(force bool) {
	// Check calibration lock before mutex lock.
	if atomic.LoadUint32(&q.c9nlock) == 1 {
		return
	}

	q.mux.Lock()
	defer func() {
		atomic.StoreUint32(&q.c9nlock, 0)
		q.mux.Unlock()
	}()

	atomic.StoreUint32(&q.c9nlock, 1)

	// Reset spinlock immediately to reduce amount of threads waiting for calibrate.
	atomic.StoreInt64(&q.spinlock, 0)

	rate := q.Rate()
	if q.l() != nil {
		msg := "queue #%s calibrate: rate %f, workers %d"
		if force {
			msg = "queue #%s force calibrate: rate %f, workers %d"
		}
		q.l().Printf(msg, q.k(), rate, atomic.LoadInt32(&q.workersUp))
	}

	for i := q.c().WorkersMax - 1; i >= q.c().WorkersMin; i-- {
		if q.workers[i].getStatus() == WorkerStatusSleep && q.workers[i].sleptEnough() {
			q.workers[i].signal(sigStop)
		}
	}

	// todo check schedID

	switch {
	case rate == 0 && q.getStatus() == StatusClose:
		for i := uint32(0); i < q.c().WorkersMax; i++ {
			if ws := q.workers[i].getStatus(); ws == WorkerStatusActive || ws == WorkerStatusSleep {
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
			if ws == WorkerStatusIdle {
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
		var target, c int32
		if target = q.getWorkersUp() / 2; target == 0 {
			target = 1
		}
		for i := q.c().WorkersMax - 1; i >= q.c().WorkersMin; i-- {
			if q.workers[i].getStatus() == WorkerStatusActive {
				q.workers[i].signal(sigSleep)
				c++
				if uint32(atomic.AddInt32(&q.workersUp, -1)) == q.c().WorkersMin || c == target {
					break
				}
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

func (q *Queue) rtWorkersMax() uint32 {
	sched, conf := uint32(0), q.c().WorkersMax
	if q.c().Schedule != nil {
		sched = q.c().Schedule.workersMax()
	}
	if sched > conf {
		return sched
	}
	return conf
}

func (q *Queue) rtParams() (workersMin, workersMax uint32, wakeupFactor, sleepFactor float32, schedID int) {
	c := q.c()
	if c.Schedule != nil {
		if workersMin, workersMax, wakeupFactor, sleepFactor, schedID = c.Schedule.Get(); schedID != -1 {
			if wakeupFactor == 0 {
				wakeupFactor = c.WakeupFactor
			}
			if sleepFactor == 0 {
				sleepFactor = c.SleepFactor
			}
			return
		}
	}
	workersMin, workersMax, wakeupFactor, sleepFactor, schedID = c.WorkersMin, c.WorkersMax, c.WakeupFactor, c.SleepFactor, -1
	return
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
	out.FullnessRate = q.Rate()

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

func (q *Queue) k() string {
	return q.config.Key
}

func (q *Queue) m() MetricsWriter {
	return q.config.MetricsWriter
}

func (q *Queue) l() Logger {
	return q.config.Logger
}
