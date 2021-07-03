package blqueue

import (
	"time"
)

const (
	defaultWakeupFactor = .75
	defaultSleepFactor  = .5
	defaultHeartbeat    = time.Second
	defaultSleepTimeout = time.Second * 5

	VerboseNone VerbosityLevel = iota
	VerboseInfo
	VerboseWarn
	VerboseError
)

type VerbosityLevel uint

type Config struct {
	Size      uint64        `json:"size"`
	Workers   uint32        `json:"workers"`
	Heartbeat time.Duration `json:"heartbeat"`

	WorkersMin   uint32  `json:"workers_min"`
	WorkersMax   uint32  `json:"workers_max"`
	WakeupFactor float32 `json:"wakeup_factor"`
	SleepFactor  float32 `json:"sleep_factor"`
	SleepTimeout time.Duration

	DequeueHandler Dequeuer
	LeakyHandler   Leaker

	MetricsKey     string `json:"metrics_key"`
	MetricsHandler MetricsWriter

	Logger         Logger
	VerbosityLevel VerbosityLevel
}

func (c *Config) Verbose(level VerbosityLevel) bool {
	return c.VerbosityLevel&level != 0
}
