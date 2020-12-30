package queue

type DummyMetrics struct{}

func (m *DummyMetrics) WorkerSleep(_ uint32) {}

func (m *DummyMetrics) WorkerWakeup(_ uint32) {}

func (m *DummyMetrics) WorkerStop(_ uint32) {}

func (m *DummyMetrics) QueuePut() {}

func (m *DummyMetrics) QueuePull() {}

func (m *DummyMetrics) QueueLeak() {}

func DummyProc(x interface{}) {
	_ = x
}
