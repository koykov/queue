package victoria

import (
	"time"

	"github.com/koykov/vmchain"
)

type Writer interface {
	WorkerSetup(active, sleep, stop uint)
	WorkerInit(idx uint32)
	WorkerSleep(idx uint32)
	WorkerWakeup(idx uint32)
	WorkerWait(idx uint32, dur time.Duration)
	WorkerStop(idx uint32, force bool, status string)
	QueuePut()
	QueuePull()
	QueueRetry(delay time.Duration)
	QueueLeak(direction string)
	QueueDeadline()
	QueueLost()
	SubqPut(subq string)
	SubqPull(subq string)
	SubqLeak(subq string)
}

// writer is a VictoriaMetrics implementation of queue.MetricsWriter.
type writer struct {
	name string
	prec time.Duration
}

// NewWriter makes a new instance of metrics writer.
func NewWriter(name string, options ...Option) Writer {
	mw := &writer{name: name}
	for _, fn := range options {
		fn(mw)
	}
	if mw.prec == 0 {
		mw.prec = time.Nanosecond
	}
	return mw
}

func (w writer) WorkerSetup(active, sleep, stop uint) {
	vmchain.Gauge("queue_workers_active", nil).WithLabel("queue", w.name).Set(float64(active))
	vmchain.Gauge("queue_workers_sleep", nil).WithLabel("queue", w.name).Set(float64(sleep))
	vmchain.Gauge("queue_workers_idle", nil).WithLabel("queue", w.name).Set(float64(stop))
}

func (w writer) WorkerInit(_ uint32) {
	vmchain.Gauge("queue_workers_active", nil).WithLabel("queue", w.name).Inc()
	vmchain.Gauge("queue_workers_idle", nil).WithLabel("queue", w.name).Dec()
}

func (w writer) WorkerSleep(_ uint32) {
	vmchain.Gauge("queue_workers_sleep", nil).WithLabel("queue", w.name).Inc()
	vmchain.Gauge("queue_workers_active", nil).WithLabel("queue", w.name).Dec()
}

func (w writer) WorkerWakeup(_ uint32) {
	vmchain.Gauge("queue_workers_active", nil).WithLabel("queue", w.name).Inc()
	vmchain.Gauge("queue_workers_sleep", nil).WithLabel("queue", w.name).Dec()
}

func (w writer) WorkerWait(_ uint32, delay time.Duration) {
	vmchain.Histogram("queue_wait").WithLabel("queue", w.name).Update(float64(delay.Nanoseconds() / int64(w.prec)))
}

func (w writer) WorkerStop(_ uint32, force bool, status string) {
	vmchain.Gauge("queue_workers_idle", nil).WithLabel("queue", w.name).Inc()
	if force {
		switch status {
		case "active":
			vmchain.Gauge("queue_workers_active", nil).WithLabel("queue", w.name).Dec()
		case "sleep":
			vmchain.Gauge("queue_workers_sleep", nil).WithLabel("queue", w.name).Dec()
		}
	} else {
		vmchain.Gauge("queue_workers_sleep", nil).WithLabel("queue", w.name).Dec()
	}
}

func (w writer) QueuePut() {
	vmchain.Counter("queue_in").WithLabel("queue", w.name).Inc()
	vmchain.Gauge("queue_size", nil).WithLabel("queue", w.name).Inc()
}

func (w writer) QueuePull() {
	vmchain.Counter("queue_out").WithLabel("queue", w.name).Inc()
	vmchain.Gauge("queue_size", nil).WithLabel("queue", w.name).Dec()
}

func (w writer) QueueRetry(delay time.Duration) {
	vmchain.Counter("queue_retry").WithLabel("queue", w.name).Inc()
	vmchain.Histogram("queue_retry_delay").WithLabel("queue", w.name).Update(float64(delay.Nanoseconds() / int64(w.prec)))
}

func (w writer) QueueLeak(direction string) {
	vmchain.Counter("queue_leak").WithLabel("queue", w.name).WithLabel("dir", direction).Inc()
	vmchain.Gauge("queue_size", nil).WithLabel("queue", w.name).Dec()
}

func (w writer) QueueDeadline() {
	vmchain.Counter("queue_deadline").WithLabel("queue", w.name).Inc()
	vmchain.Gauge("queue_size", nil).WithLabel("queue", w.name).Dec()
}

func (w writer) QueueLost() {
	vmchain.Counter("queue_lost").WithLabel("queue", w.name).Inc()
	vmchain.Gauge("queue_size", nil).WithLabel("queue", w.name).Dec()
}

func (w writer) SubqPut(subq string) {
	vmchain.Counter("queue_subq_in").WithLabel("queue", w.name).WithLabel("subq", subq).Inc()
	vmchain.Gauge("queue_subq_size", nil).WithLabel("queue", w.name).WithLabel("subq", subq).Inc()
}

func (w writer) SubqPull(subq string) {
	vmchain.Counter("queue_subq_out").WithLabel("queue", w.name).WithLabel("subq", subq).Inc()
	vmchain.Gauge("queue_subq_size", nil).WithLabel("queue", w.name).WithLabel("subq", subq).Dec()
}

func (w writer) SubqLeak(subq string) {
	vmchain.Counter("queue_subq_leak").WithLabel("queue", w.name).WithLabel("subq", subq).Inc()
	vmchain.Gauge("queue_subq_size", nil).WithLabel("queue", w.name).WithLabel("subq", subq).Dec()
}

var _ = NewWriter
