package queue

import (
	"time"

	"github.com/koykov/queue/qos"
)

const (
	// Queue default fullness rate to wake sleep workers.
	defaultWakeupFactor = .75
	// Queue default fullness rate to sleep redundant active workers.
	defaultSleepFactor = .5
	// Queue default heartbeat rate.
	defaultHeartbeatInterval = time.Second
	// Worker default sleep interval.
	// After that interval slept worker will stop.
	defaultSleepInterval = time.Second * 5
	// Default simultaneous enqueue operation limit to start force calibration.
	defaultForceCalibrationLimit = 1000
	// Default top limit of factors.
	defaultFactorLimit = .999999
)

// Config describes queue properties and behavior.
type Config struct {
	// Queue capacity.
	// Mandatory param if QoS config omitted. QoS (if provided) summing capacity will overwrite this field.
	Capacity uint64
	// Streams allows to avoid mutex starvation by sharing items among Streams sub-channels instead of one singe
	// channel.
	Streams uint32
	// MaxRetries determines the maximum number of item processing retries.
	// If MaxRetries is exceeded, the item will send to DLQ (if possible).
	// The initial attempt is not counted as a retry.
	MaxRetries uint32
	// Interval between retry attempts.
	RetryInterval time.Duration
	// Backoff calculates delay to next processing attempt.
	// Works only with non-empty RetryInterval.
	Backoff Backoff
	// Simultaneous enqueue operation limit to start force calibration.
	// Works only on balanced queues.
	// If this param omit defaultForceCalibrationLimit (1000) will use instead.
	ForceCalibrationLimit uint32
	// Workers number.
	// Setting this param disables balancing feature. If you want to have balancing use params WorkersMin and WorkersMax
	// instead.
	Workers uint32
	// HeartbeatInterval rate interval. Need to perform service operation like queue calibration, workers handling, etc.
	// Setting this param too big (greater than 1 second) is counterproductive - the queue will rarely calibrate and
	// result may be insufficient good.
	// If this param omit defaultHeartbeatInterval (1 second) will use instead.
	HeartbeatInterval time.Duration

	// QoS scheduling settings.
	// If this param omit FIFO queue will init by default.
	// See qos/config.go
	QoS *qos.Config

	// Minimum workers number.
	// Setting this param less than WorkersMax enables balancing feature.
	WorkersMin uint32
	// Maximum workers number.
	// Setting this param greater than WorkersMin enables balancing feature.
	WorkersMax uint32
	// Worker wake up factor in dependency of queue fullness rate.
	// When queue fullness rate will exceed that factor, then first available slept worker will wake.
	// WakeupFactor must be in range [0..0.999999].
	// If this param omit defaultWakeupFactor (0.75) will use instead.
	WakeupFactor float32
	// Worker sleep factor in dependency of queue fullness rate.
	// When queue fullness rate will less than  that factor, one of active workers will put to sleep.
	// SleepFactor must be in range [0..0.999999].
	// If this param omit defaultSleepFactor (0.5) will use instead.
	SleepFactor float32
	// Limit of workers could send to sleep at once.
	// If this param omit the half of available workers will send to sleep at calibration.
	SleepThreshold uint32
	// How long slept worker will wait until stop.
	// If this param omit defaultSleepInterval (5 seconds) will use instead.
	SleepInterval time.Duration

	// Schedule contains base params (like workers min/max and factors) for specific time ranges.
	// See schedule.go for usage examples.
	Schedule *Schedule

	// Worker represents queue worker.
	// Mandatory param.
	Worker Worker
	// Dead letter queue to catch leaky items.
	// Setting this param enables leaky feature.
	DLQ Enqueuer
	// Put failed items to DLQ.
	// Better to use together with MaxRetries. After all processing attempts item will send to DLQ.
	FailToDLQ bool
	// Put denied by deadline items to DLQ.
	DeadlineToDLQ bool
	// LeakDirection indicates queue side to leak items (rear or front).
	LeakDirection LeakDirection
	// FrontLeakAttempts indicates how many times queue may be shifted to free up space for new rear item.
	// On limit overflow rear direction will use by fallback.
	// Low values required.
	// If this param omit defaultFrontLeakAttempts (5) will use instead.
	FrontLeakAttempts uint32

	// DelayInterval between item enqueue and processing.
	// Settings this param enables delayed execution (DE) feature.
	// DE guarantees that item will processed by worker after at least DelayInterval time.
	// The opposite param to DeadlineInterval.
	DelayInterval time.Duration
	// DeadlineInterval to skip useless item processing.
	// Setting this param enables Deadline-Aware Queue (DAQ) feature.
	// DAQ guarantees that item will not process if time is over when worker takes it from queue.
	// The opposite param to DelayInterval.
	DeadlineInterval time.Duration

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
	if c.QoS != nil {
		cpy.QoS = c.QoS.Copy()
	}
	return &cpy
}
