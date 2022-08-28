package blqueue

import "time"

// DummyMetrics is a stub metrics writer handler that uses by default and does nothing.
// Need just to reduce checks in code.
type DummyMetrics struct{}

func (DummyMetrics) WorkerSetup(_ string, _, _, _ uint)                    {}
func (DummyMetrics) WorkerInit(_ string, _ uint32)                         {}
func (DummyMetrics) WorkerSleep(_ string, _ uint32)                        {}
func (DummyMetrics) WorkerWakeup(_ string, _ uint32)                       {}
func (DummyMetrics) WorkerWait(_ string, _ uint32, _ time.Duration)        {}
func (DummyMetrics) WorkerStop(_ string, _ uint32, _ bool, _ WorkerStatus) {}
func (DummyMetrics) QueuePut(_ string)                                     {}
func (DummyMetrics) QueuePull(_ string)                                    {}
func (DummyMetrics) QueueRetry(_ string)                                   {}
func (DummyMetrics) QueueLeak(_ string)                                    {}
func (DummyMetrics) QueueLost(_ string)                                    {}
