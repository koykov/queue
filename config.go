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

	// Logger verbosity levels.
	VerboseNone VerbosityLevel = iota
	VerboseInfo
	VerboseWarn
	VerboseError
)

type VerbosityLevel uint

// Queue config.
type Config struct {
	// Queue capacity.
	Size uint64 `json:"size"`
	// Workers number.
	// Setting this param disables balancing feature. If you want to have balancing use params WorkersMin/WorkersMax
	// instead.
	Workers uint32 `json:"workers"`
	// Heartbeat rate interval. Need to perform service operation like queue rebalance, workers handling, etc.
	Heartbeat time.Duration `json:"heartbeat"`

	// Minimum workers number.
	// Setting this param less than WorkersMax enables balancing feature.
	WorkersMin uint32 `json:"workers_min"`
	// Maximum workers number.
	// Setting this param greater than WorkersMin enables balancing feature.
	WorkersMax uint32 `json:"workers_max"`
	// Worker wake up factor in dependency of queue fullness rate.
	// When queue fullness rate will exceeds that factor, then first available slept worker will wake.
	WakeupFactor float32 `json:"wakeup_factor"`
	// Worker sleep factor in dependency of queue fullness rate.
	// When queue fullness rate will less than  that factor, one of active workers will put to sleep.
	SleepFactor float32 `json:"sleep_factor"`
	// How long slept worker will wait until stop.
	SleepTimeout time.Duration

	// Dequeue handler for workers.
	Dequeuer Dequeuer
	// Handler to catch leaky entries.
	// Setting this param enables leaky feature.
	Catcher Catcher

	// Queue key in metrics.
	MetricsKey string `json:"metrics_key"`
	// Metrics writer handler.
	MetricsWriter MetricsWriter

	// Logger handler.
	Logger Logger
	// Verbosity level of logger.
	VerbosityLevel VerbosityLevel
}

// Copy config instance to protect queue from changing params in runtime.
func (c *Config) Copy() *Config {
	cpy := *c
	return &cpy
}

// Check verbosity level.
func (c *Config) Verbose(level VerbosityLevel) bool {
	return c.VerbosityLevel&level != 0
}
