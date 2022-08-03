package blqueue

import (
	"time"
)

const (
	// Queue default fullness rate to wake sleep workers.
	defaultWakeupFactor = .75
	// Queue default fullness rate to sleep redundant active workers.
	defaultSleepFactor = .5
	// Queue default heartbeat rate.
	defaultHeartbeat = time.Second
	// Worker default sleep interval.
	// After that interval slept worker will stop.
	defaultSleepTimeout = time.Second * 5
	// Default simultaneous enqueue operation limit to start force calibration.
	defaultForceCalibrationLimit = 1000
)

// Config describes queue properties and behavior.
type Config struct {
	// Unique queue key. Indicates queue in logs and metrics.
	// Mandatory param.
	Key string
	// Queue capacity.
	// Mandatory param.
	Size uint64
	// MaxRetries determines the maximum number of item processing retries.
	// If MaxRetries is exceeded, the item will send to DLQ (if possible).
	// The initial attempt is not counted as a retry.
	MaxRetries uint32
	// Simultaneous enqueue operation limit to start force calibration.
	// Works only on balanced queues.
	// If this param omit defaultForceCalibrationLimit (1000) will use instead.
	ForceCalibrationLimit uint32
	// Workers number.
	// Setting this param disables balancing feature. If you want to have balancing use params WorkersMin and WorkersMax
	// instead.
	Workers uint32
	// Heartbeat rate interval. Need to perform service operation like queue calibration, workers handling, etc.
	// Setting this param too big (greater than 1 second) is counterproductive - the queue will rarely calibrate and
	// result may be insufficient good.
	// If this param omit defaultHeartbeat (1 second) will use instead.
	Heartbeat time.Duration

	// Minimum workers number.
	// Setting this param less than WorkersMax enables balancing feature.
	WorkersMin uint32
	// Maximum workers number.
	// Setting this param greater than WorkersMin enables balancing feature.
	WorkersMax uint32
	// Worker wake up factor in dependency of queue fullness rate.
	// When queue fullness rate will exceed that factor, then first available slept worker will wake.
	// If this param omit defaultWakeupFactor (0.75) will use instead.
	WakeupFactor float32
	// Worker sleep factor in dependency of queue fullness rate.
	// When queue fullness rate will less than  that factor, one of active workers will put to sleep.
	// If this param omit defaultSleepFactor (0.5) will use instead.
	SleepFactor float32
	// How long slept worker will wait until stop.
	// If this param omit defaultSleepTimeout (5 seconds) will use instead.
	SleepTimeout time.Duration

	// Schedule contains base params (like workers min/max and factors) for specific time ranges.
	// See schedule.go for usage examples.
	Schedule *Schedule

	// Dequeuer is a worker's dequeue helper.
	// Mandatory param.
	Dequeuer Dequeuer
	// Dead letter queue to catch leaky items.
	// Setting this param enables leaky feature.
	DLQ DLQ
	// Put failed items to DLQ.
	// Better to use together with MaxRetries. After all processing attempts item will send to DLQ.
	FailToDLQ bool

	// Delay between item enqueue and processing.
	// Settings this param enables delayed execution (DE) feature.
	// DE guarantees that item will processed by worker after at least Delay time.
	Delay time.Duration

	// Clock represents clock keeper.
	// If this param omit nativeClock will use instead (see clock.go).
	Clock Clock

	// Metrics writer handler.
	MetricsWriter MetricsWriter

	// Logger handler.
	Logger Logger
}

// Copy copies config instance to protect queue from changing params after start.
// It means that after starting queue all config modifications will have no effect.
func (c *Config) Copy() *Config {
	cpy := *c
	if c.Schedule != nil {
		cpy.Schedule = c.Schedule.Copy()
	}
	return &cpy
}
