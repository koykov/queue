package main

import "time"

type RequestInit struct {
	Size      uint64        `json:"size"`
	Workers   uint32        `json:"workers"`
	Heartbeat time.Duration `json:"heartbeat"`

	WorkersMin   uint32  `json:"workers_min"`
	WorkersMax   uint32  `json:"workers_max"`
	WakeupFactor float32 `json:"wakeup_factor"`
	SleepFactor  float32 `json:"sleep_factor"`

	MetricsKey string `json:"metrics_key"`

	ProducersMin uint32 `json:"producers_min"`
	ProducersMax uint32 `json:"producers_max"`

	AllowLeak bool `json:"allow_leak"`
}
