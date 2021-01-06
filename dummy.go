package queue

import "time"

type DummyMetrics struct{}

func (m *DummyMetrics) WorkerSetup(_, _, _ uint) {}

func (m *DummyMetrics) WorkerSleep(_ uint32) {}

func (m *DummyMetrics) WorkerWakeup(_ uint32) {}

func (m *DummyMetrics) WorkerStop(_ uint32) {}

func (m *DummyMetrics) QueuePut() {}

func (m *DummyMetrics) QueuePull() {}

func (m *DummyMetrics) QueueLeak() {}

func DummyProc(x interface{}) {
	_ = x
	time.Sleep(time.Nanosecond * 75)
}

type DummyLeak struct{}

func (l *DummyLeak) Catch(x interface{}) {
	_ = x
}
