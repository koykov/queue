package queue

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

// Queue is an implementation of balanced leaky queue.
//
// The queue balances among available workers [Config.WorkersMin...Config.WorkersMax] in realtime.
// Queue also has leaky feature: when queue is full and new items continue to flow, then leaked items will forward to
// DLQ (dead letter queue).
// For queues with variadic daily load exists special scheduler (see schedule.go) that allow to specify variadic queue
// params for certain time ranges.
type Queue struct {
	bitset.Bitset
	// Config instance.
	config *Config
	// ID of actual schedule rule. Contains -1 by default (no rule found).
	schedID int
	// The number of maximum workers that queue may contain considering all schedule rules and config params.
	wmax uint32

	// Actual queue status.
	status Status
	// Internal items stream.
	stream stream

	mux sync.Mutex
	// Workers pool.
	workers []*worker

	once sync.Once

	// Counter of active workers.
	workersUp int32
	// Calibration lock counter.
	c9nlock uint32
	// Spinlock of queue.
	spinlock int64
	// Enqueue lock counter.
	enqlock int64

	Err error
}

// item is a wrapper for queue element with retries count.
type item struct {
	payload interface{}
	retries uint32
	dexpire int64 // Delayed execution expire time (Unix ns timestamp).
}

// Items stream.
type stream chan item

// realtimeParams describes queue params for current time.
type realtimeParams struct {
	WorkersMin, WorkersMax    uint32
	WakeupFactor, SleepFactor float32
}

// New makes new queue instance and initialize it according config params.
func New(config *Config) (*Queue, error) {
	q := &Queue{
		// Make a copy of config instance to protect queue from changing params after start.
		config: config.Copy(),
	}
	q.once.Do(q.init)
	return q, q.Err
}

// Init queue.
func (q *Queue) init() {
	c := q.config

	// Check mandatory params.
	if c.Capacity == 0 {
		q.Err = ErrNoCapacity
		q.status = StatusFail
		return
	}
	if c.Worker == nil {
		q.Err = ErrNoWorker
		q.status = StatusFail
		return
	}

	if c.Clock == nil {
		c.Clock = nativeClock{}
	}

	if c.MetricsWriter == nil {
		// Use dummy MW.
		c.MetricsWriter = DummyMetrics{}
	}

	// Check workers numbers params.
	if c.Workers > 0 && c.WorkersMin == 0 {
		c.WorkersMin = c.Workers
	}
	if c.Workers > 0 && c.WorkersMax == 0 {
		c.WorkersMax = c.Workers
	}
	if c.WorkersMax < c.WorkersMin {
		c.WorkersMin = c.WorkersMax
	}
	if c.WorkersMax == 0 {
		q.Err = ErrNoWorkers
		q.status = StatusFail
		return
	}

	// Check non-mandatory params and set default values if needed.
	if c.ForceCalibrationLimit == 0 {
		c.ForceCalibrationLimit = defaultForceCalibrationLimit
	}
	if c.WakeupFactor <= 0 {
		c.WakeupFactor = defaultWakeupFactor
	}
	if c.WakeupFactor > defaultFactorLimit {
		c.WakeupFactor = defaultFactorLimit
	}

	if c.SleepFactor <= 0 {
		c.SleepFactor = defaultSleepFactor
	}
	if c.SleepFactor > defaultFactorLimit {
		c.SleepFactor = defaultFactorLimit
	}

	if c.WakeupFactor < c.SleepFactor {
		c.WakeupFactor = c.SleepFactor
	}

	if c.SleepInterval == 0 {
		c.SleepInterval = defaultSleepInterval
	}
	if c.HeartbeatInterval == 0 {
		c.HeartbeatInterval = defaultHeartbeatInterval
	}

	// Create the stream.
	q.stream = make(stream, c.Capacity)

	// Check flags.
	q.SetBit(flagBalanced, c.WorkersMin < c.WorkersMax || c.Schedule != nil)
	q.SetBit(flagLeaky, c.DLQ != nil)

	// Check initial params.
	q.wmax = q.workersMaxDaily()
	var params realtimeParams
	params, q.schedID = q.rtParams()

	// Make workers pool/
	q.workers = make([]*worker, q.wmax)
	var i uint32
	for i = 0; i < q.wmax; i++ {
		q.mw().WorkerSleep(i)
		q.workers[i] = makeWorker(i, c)
	}
	q.mw().WorkerSetup(0, 0, uint(params.WorkersMax))

	// Start [0...workersMin] workers.
	for i = 0; i < params.WorkersMin; i++ {
		q.workers[i].signal(sigInit)
		go q.workers[i].await(q)
	}
	q.workersUp = int32(params.WorkersMin)

	if q.CheckBit(flagBalanced) {
		// Init background heartbeat ticker.
		tickerHB := time.NewTicker(c.HeartbeatInterval)
		go func() {
			for {
				select {
				case <-tickerHB.C:
					// Calibrate queue on each tick in regular mode.
					q.calibrate(false)
					if q.Rate() == 0 && q.getStatus() == StatusClose {
						tickerHB.Stop()
						// Exit on empty stopped queue.
						return
					}
				}
			}
		}()
	}

	// Queue is ready!
	q.setStatus(StatusActive)
}

// Enqueue puts x to the queue.
func (q *Queue) Enqueue(x interface{}) error {
	q.once.Do(q.init)
	// Check if enqueue is possible.
	if status := q.getStatus(); status == StatusClose || status == StatusFail {
		return ErrQueueClosed
	}

	atomic.AddInt64(&q.enqlock, 1)
	defer atomic.AddInt64(&q.enqlock, -1)

	if q.CheckBit(flagBalanced) {
		defer atomic.AddInt64(&q.spinlock, -1)
		// Consider spinlock and calibration limit on balanced queue.
		if atomic.AddInt64(&q.spinlock, 1) >= int64(q.c().ForceCalibrationLimit) {
			q.calibrate(true)
		}
	}
	itm := item{payload: x}
	if di := q.c().DelayInterval; di > 0 {
		itm.dexpire = q.clk().Now().Add(di).UnixNano()
	}
	return q.renqueue(&itm)
}

// Put wrapped item to the queue.
// This method also uses for enqueue retries (see Config.MaxRetries).
func (q *Queue) renqueue(itm *item) error {
	q.mw().QueuePut()
	if q.CheckBit(flagLeaky) {
		// Put item to the stream in leaky mode.
		select {
		case q.stream <- *itm:
			return nil
		default:
			// Leak the item to DLQ.
			err := q.c().DLQ.Enqueue(itm.payload)
			q.mw().QueueLeak()
			return err
		}
	} else {
		// Regular put (blocking mode).
		q.stream <- *itm
		return nil
	}
}

// Size return actual size of the queue.
func (q *Queue) Size() int {
	return len(q.stream)
}

// Capacity return max size of the queue.
func (q *Queue) Capacity() int {
	return cap(q.stream)
}

// Rate returns size to capacity ratio.
func (q *Queue) Rate() float32 {
	return float32(len(q.stream)) / float32(cap(q.stream))
}

// Close gracefully stops the queue.
//
// After receiving of close signal at least workersMin number of workers will work so long as queue has items.
// Enqueue of new items to queue will forbid.
func (q *Queue) Close() error {
	return q.close(false)
}

// ForceClose closes the queue and immediately stops all active and sleeping workers.
//
// Remaining items in the queue will throw to the trash.
func (q *Queue) ForceClose() error {
	return q.close(true)
}

func (q *Queue) close(force bool) error {
	if q.getStatus() == StatusClose {
		return ErrQueueClosed
	}
	if q.l() != nil {
		msg := "caught close signal"
		if force {
			msg = "caught force close signal"
		}
		q.l().Printf(msg)
	}
	// Set the status.
	q.setStatus(StatusClose)
	// Wait till all enqueue operations will finish.
	for atomic.LoadInt64(&q.enqlock) > 0 {
	}
	// Close the stream.
	// Please note, this is not the end for regular close case. Workers continue works while queue has items.
	close(q.stream)

	if force {
		// Immediately stop all active/sleeping workers.
		q.mux.Lock()
		for i := int(q.wmax - 1); i >= 0; i-- {
			switch q.workers[i].getStatus() {
			case WorkerStatusActive:
				q.workers[i].signal(sigForceStop)
				atomic.AddInt32(&q.workersUp, -1)
			case WorkerStatusSleep:
				q.workers[i].signal(sigForceStop)
			}
		}
		q.mux.Unlock()
		// Throw all remaining items to DLQ or trash.
		for len(q.stream) > 0 {
			itm := <-q.stream
			if q.CheckBit(flagLeaky) {
				_ = q.c().DLQ.Enqueue(itm.payload)
				q.mw().QueueLeak()
			} else {
				q.mw().QueueLost()
			}
		}
	}
	return nil
}

// Internal calibration helper.
func (q *Queue) calibrate(force bool) {
	// Check calibration lock before mutex lock.
	if atomic.LoadUint32(&q.c9nlock) == 1 {
		// Calibration is busy.
		return
	}

	q.mux.Lock()
	defer func() {
		// Release calibration lock.
		atomic.StoreUint32(&q.c9nlock, 0)
		q.mux.Unlock()
	}()

	// Calibration is acquired.
	atomic.StoreUint32(&q.c9nlock, 1)

	// Reset spinlock immediately to reduce amount of threads waiting for calibrate.
	atomic.StoreInt64(&q.spinlock, 0)

	rate := q.Rate()
	if q.l() != nil {
		msg := "calibrate: rate %f, workers %d"
		if force {
			msg = "force calibrate: rate %f, workers %d"
		}
		q.l().Printf(msg, rate, atomic.LoadInt32(&q.workersUp))
	}

	// Check and stop pre-sleeping workers.
	for i := q.c().WorkersMax - 1; i >= q.c().WorkersMin; i-- {
		if q.workers[i].getStatus() == WorkerStatusSleep && q.workers[i].sleptEnough() {
			q.workers[i].signal(sigStop)
		}
	}

	// Check schedID change.
	var (
		params  realtimeParams
		schedID int
	)
	if params, schedID = q.rtParams(); schedID != q.schedID {
		q.schedID = schedID
		if q.l() != nil {
			q.l().Printf("switch to schedID %d (workers %d/%d, wakeup factor %f, sleep factor %f)",
				schedID, params.WorkersMin, params.WorkersMax, params.WakeupFactor, params.SleepFactor)
		}
		// Stop all workers in range [workersMax...wmax].
		// wmax is a number of maximum workers queue may have.
		// workersMax is a maximum number of workers queue may have in current time range.
		if q.wmax > params.WorkersMax {
			for i := q.wmax - 1; i >= params.WorkersMax; i-- {
				if q.workers[i].getStatus() == WorkerStatusActive {
					q.workers[i].stop(true)
					atomic.AddInt32(&q.workersUp, -1)
				}
			}
		}
		// Check new params.WorkersMin exceeds number of active workers.
		if wu := uint32(q.getWorkersUp()); params.WorkersMin > wu {
			// Start params.WorkersMin-workersUp workers to satisfy queue.
			target := params.WorkersMin - wu
			var c uint32
			for i := uint32(0); i < q.wmax; i++ {
				switch q.workers[i].getStatus() {
				case WorkerStatusIdle:
					q.workers[i].signal(sigInit)
					go q.workers[i].await(q)
				case WorkerStatusSleep:
					q.workers[i].signal(sigWakeup)
				default:
					continue
				}
				c++
				atomic.AddInt32(&q.workersUp, 1)
				if c == target {
					break
				}
			}
		}
		// Calculate actual numbers of active, sleeping and idle workers.
		var active, sleep, idle uint
		for i := uint32(0); i < params.WorkersMax; i++ {
			switch q.workers[i].getStatus() {
			case WorkerStatusIdle:
				idle++
			case WorkerStatusSleep:
				sleep++
			case WorkerStatusActive:
				active++
			}
		}
		// Reinitialize workers counters in metrics.
		q.mw().WorkerSetup(active, sleep, idle)
	}

	// Calibration issues.
	switch {
	case rate == 0 && q.getStatus() == StatusClose:
		// Queue is closed and empty. Force stops all active or sleeping workers.
		for i := uint32(0); i < params.WorkersMax; i++ {
			if ws := q.workers[i].getStatus(); ws == WorkerStatusActive || ws == WorkerStatusSleep {
				q.workers[i].signal(sigForceStop)
			}
		}
	case rate >= params.WakeupFactor:
		// Queue fullness rate exceeds wakeupFactor. Need to start first available idle or sleeping worker.
		if uint32(q.getWorkersUp()) == params.WorkersMax {
			return
		}
		for i := params.WorkersMin; i < params.WorkersMax; i++ {
			ws := q.workers[i].getStatus()
			if ws == WorkerStatusActive {
				continue
			}
			if ws == WorkerStatusIdle {
				q.workers[i].signal(sigInit)
				go q.workers[i].await(q)
			} else {
				q.workers[i].signal(sigWakeup)
			}
			atomic.AddInt32(&q.workersUp, 1)
			// By default, only one worker starts at once. That's why need to keep heartbeat param enough small (<=1s).
			break
		}
	case rate <= params.SleepFactor:
		// Queue fullness rate fell less than sleep factor. So need to put worker(-s) to sleep.
		if uint32(q.getWorkersUp()) == params.WorkersMin {
			return
		}
		var target, c int32
		// Workers put to sleep by chunks of workersUp / 2.
		if target = q.getWorkersUp() / 2; target == 0 {
			target = 1
		}
		for i := params.WorkersMax - 1; i >= params.WorkersMin; i-- {
			if q.workers[i].getStatus() == WorkerStatusActive {
				q.workers[i].signal(sigSleep)
				c++
				if uint32(atomic.AddInt32(&q.workersUp, -1)) == params.WorkersMin || c == target {
					break
				}
			}
		}
	case rate == 1:
		// Queue is full and throttled.
		q.setStatus(StatusThrottle)
	default:
		// Restore active status after throttle.
		if q.getStatus() == StatusThrottle {
			q.setStatus(StatusActive)
		}
	}
}

// Get number maximum workers that queue may contain considering all schedule rules and config params.
func (q *Queue) workersMaxDaily() uint32 {
	sched, conf := uint32(0), q.c().WorkersMax
	if q.c().Schedule != nil {
		sched = q.c().Schedule.WorkersMaxDaily()
	}
	if sched > conf {
		return sched
	}
	return conf
}

// Get realtime queue params according schedule rules.
func (q *Queue) rtParams() (params realtimeParams, schedID int) {
	c := q.c()
	if c.Schedule != nil {
		var schedParams ScheduleParams
		if schedParams, schedID = c.Schedule.Get(); schedID != -1 {
			params = realtimeParams(schedParams)
			if params.WakeupFactor == 0 {
				params.WakeupFactor = c.WakeupFactor
			}
			if params.SleepFactor == 0 {
				params.SleepFactor = c.SleepFactor
			}
			return
		}
	}
	schedID = -1
	params.WorkersMin = c.WorkersMin
	params.WorkersMax = c.WorkersMax
	params.WakeupFactor = c.WakeupFactor
	params.SleepFactor = c.SleepFactor
	return
}

// Get number of active workers.
func (q *Queue) getWorkersUp() int32 {
	return atomic.LoadInt32(&q.workersUp)
}

// Set status of the queue.
func (q *Queue) setStatus(status Status) {
	atomic.StoreUint32((*uint32)(&q.status), uint32(status))
}

// Get status of the queue.
func (q *Queue) getStatus() Status {
	return Status(atomic.LoadUint32((*uint32)(&q.status)))
}

func (q *Queue) String() string {
	var out = struct {
		Capacity      uint64        `json:"capacity"`
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

	out.Capacity = q.config.Capacity
	out.Workers = q.config.Workers
	out.Heartbeat = q.config.HeartbeatInterval
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

func (q *Queue) clk() Clock {
	return q.config.Clock
}

func (q *Queue) mw() MetricsWriter {
	return q.config.MetricsWriter
}

func (q *Queue) l() Logger {
	return q.config.Logger
}

var _ = New
