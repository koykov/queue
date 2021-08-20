package blqueue

type DummyMetrics struct{}

func (*DummyMetrics) WorkerSetup(_, _, _ uint)                    {}
func (*DummyMetrics) WorkerInit(_ uint32)                         {}
func (*DummyMetrics) WorkerSleep(_ uint32)                        {}
func (*DummyMetrics) WorkerWakeup(_ uint32)                       {}
func (*DummyMetrics) WorkerStop(_ uint32, _ bool, _ WorkerStatus) {}
func (*DummyMetrics) QueuePut()                                   {}
func (*DummyMetrics) QueuePull()                                  {}
func (*DummyMetrics) QueueLeak()                                  {}

type DummyCatcher struct{}

func (*DummyCatcher) Catch(_ interface{}) {}

type DummyLog struct{}

func (*DummyLog) Printf(string, ...interface{}) {}
func (*DummyLog) Print(...interface{})          {}
func (*DummyLog) Println(...interface{})        {}
