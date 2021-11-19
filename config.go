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
)

// Queue config.
type Config struct {
	// Unique queue key. Indicates queue in logs and metrics.
	Key string
	// Queue capacity.
	Size uint64
	// Workers number.
	// Setting this param disables balancing feature. If you want to have balancing use params WorkersMin/WorkersMax
	// instead.
	Workers uint32
	// Heartbeat rate interval. Need to perform service operation like queue rebalance, workers handling, etc.
	Heartbeat time.Duration

	// Minimum workers number.
	// Setting this param less than WorkersMax enables balancing feature.
	WorkersMin uint32
	// Maximum workers number.
	// Setting this param greater than WorkersMin enables balancing feature.
	WorkersMax uint32
	// Worker wake up factor in dependency of queue fullness rate.
	// When queue fullness rate will exceed that factor, then first available slept worker will wake.
	WakeupFactor float32
	// Worker sleep factor in dependency of queue fullness rate.
	// When queue fullness rate will less than  that factor, one of active workers will put to sleep.
	SleepFactor float32
	// How long slept worker will wait until stop.
	SleepTimeout time.Duration

	// Dequeue handler for workers.
	DequeueWorker DequeueWorker
	// Dead letter queue to catch leaky items.
	// Setting this param enables leaky feature.
	DLQ DLQ

	// Metrics writer handler.
	MetricsWriter MetricsWriter

	// Logger handler.
	Logger Logger
}

// Copy config instance to protect queue from changing params in runtime.
func (c *Config) Copy() *Config {
	cpy := *c
	return &cpy
}
