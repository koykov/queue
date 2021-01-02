package queue

import "time"

const (
	defaultWakeupFactor = .75
	defaultSleepFactor  = .5
	defaultHeartbeat    = time.Millisecond
)

type Config struct {
	Size      uint64 `json:"size"`
	Proc      Proc
	Workers   uint32        `json:"workers"`
	Heartbeat time.Duration `json:"heartbeat"`

	WorkersMin   uint32  `json:"workers_min"`
	WorkersMax   uint32  `json:"workers_max"`
	WakeupFactor float32 `json:"wakeup_factor"`
	SleepFactor  float32 `json:"sleep_factor"`

	LeakyHandler Leaker

	MetricsKey     string `json:"metrics_key"`
	MetricsHandler MetricsWriter
}