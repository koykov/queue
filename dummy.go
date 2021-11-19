package blqueue

type DummyMetrics struct{}

func (*DummyMetrics) WorkerSetup(_ string, _, _, _ uint)                    {}
func (*DummyMetrics) WorkerInit(_ string, _ uint32)                         {}
func (*DummyMetrics) WorkerSleep(_ string, _ uint32)                        {}
func (*DummyMetrics) WorkerWakeup(_ string, _ uint32)                       {}
func (*DummyMetrics) WorkerStop(_ string, _ uint32, _ bool, _ WorkerStatus) {}
func (*DummyMetrics) QueuePut(_ string)                                     {}
func (*DummyMetrics) QueuePull(_ string)                                    {}
func (*DummyMetrics) QueueLeak(_ string)                                    {}

type DummyDLQ struct{}

func (*DummyDLQ) Enqueue(_ interface{}) {}
