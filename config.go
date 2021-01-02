package queue

import "time"

const (
	defaultWakeupFactor = .75
	defaultSleepFactor  = .5
	defaultHeartbeat    = time.Millisecond
)

type Config struct {
	Size      uint64
	Proc      Proc
	Workers   uint32
	Heartbeat time.Duration

	WorkersMin, WorkersMax    uint32
	WakeupFactor, SleepFactor float32

	LeakyHandler Leaker

	MetricsKey     string
	MetricsHandler MetricsWriter
}
