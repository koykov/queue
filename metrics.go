package queue

type MetricsWriter interface {
	WorkerSleep(idx uint32)
	WorkerWakeup(idx uint32)
	WorkerStop(idx uint32)
	QueuePut()
	QueuePull()
	QueueLeak()
}

type DummyMetrics struct{}

func (m *DummyMetrics) WorkerSleep(_ uint32) {}

func (m *DummyMetrics) WorkerWakeup(_ uint32) {}

func (m *DummyMetrics) WorkerStop(_ uint32) {}

func (m *DummyMetrics) QueuePut() {}

func (m *DummyMetrics) QueuePull() {}

func (m *DummyMetrics) QueueLeak() {}
