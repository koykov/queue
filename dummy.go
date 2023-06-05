package queue

import "time"

// DummyMetrics is a stub metrics writer handler that uses by default and does nothing.
// Need just to reduce checks in code.
type DummyMetrics struct{}

func (DummyMetrics) WorkerSetup(_, _, _ uint)                    {}
func (DummyMetrics) WorkerInit(_ uint32)                         {}
func (DummyMetrics) WorkerSleep(_ uint32)                        {}
func (DummyMetrics) WorkerWakeup(_ uint32)                       {}
func (DummyMetrics) WorkerWait(_ uint32, _ time.Duration)        {}
func (DummyMetrics) WorkerStop(_ uint32, _ bool, _ WorkerStatus) {}
func (DummyMetrics) QueuePut()                                   {}
func (DummyMetrics) QueuePull()                                  {}
func (DummyMetrics) QueueRetry()                                 {}
func (DummyMetrics) QueueLeak(_ LeakDirection)                   {}
func (DummyMetrics) QueueDeadline()                              {}
func (DummyMetrics) QueueLost()                                  {}
func (DummyMetrics) SubqPut(_ string)                            {}
func (DummyMetrics) SubqPull(_ string)                           {}
func (DummyMetrics) SubqLeak(_ string)                           {}

// DummyDLQ is a stub DLQ implementation. It does nothing and need for queues with leak tolerance.
// It just leaks data to the trash.
type DummyDLQ struct{}

func (DummyDLQ) Enqueue(_ any) error { return nil }
func (DummyDLQ) Size() int           { return 0 }
func (DummyDLQ) Capacity() int       { return 0 }
func (DummyDLQ) Rate() float32       { return 0 }
func (DummyDLQ) Close() error        { return nil }
