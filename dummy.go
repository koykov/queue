package blqueue

import "time"

type DummyMetrics struct{}

func (*DummyMetrics) WorkerSetup(_, _, _ uint)                    {}
func (*DummyMetrics) WorkerInit(_ uint32)                         {}
func (*DummyMetrics) WorkerSleep(_ uint32)                        {}
func (*DummyMetrics) WorkerWakeup(_ uint32)                       {}
func (*DummyMetrics) WorkerStop(_ uint32, _ bool, _ WorkerStatus) {}
func (*DummyMetrics) QueuePut()                                   {}
func (*DummyMetrics) QueuePull()                                  {}
func (*DummyMetrics) QueueLeak()                                  {}

type DummyLeak struct{}

func (*DummyLeak) Catch(x interface{}) {
	_ = x
}

type DummyLog struct{}

func (*DummyLog) Printf(string, ...interface{}) {}
func (*DummyLog) Print(...interface{})          {}
func (*DummyLog) Println(...interface{})        {}
func (*DummyLog) Fatal(...interface{})          {}
func (*DummyLog) Fatalf(string, ...interface{}) {}
func (*DummyLog) Fatalln(...interface{})        {}

func DummyProc(x interface{}) {
	_ = x
	time.Sleep(time.Nanosecond * 75)
}
