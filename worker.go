package blqueue

import (
	"sync/atomic"
	"time"
)

type WorkerStatus uint32
type signal uint32

const (
	WorkerStatusIdle WorkerStatus = iota
	WorkerStatusActive
	WorkerStatusSleep

	signalInit signal = iota
	signalSleep
	signalWakeup
	signalStop
	signalForceStop
)

type ctl chan signal

type worker struct {
	idx    uint32
	status WorkerStatus
	ctl    ctl
	lastTS time.Time
	proc   Dequeuer
	config *Config
}

func makeWorker(idx uint32, config *Config) *worker {
	w := &worker{
		idx:    idx,
		status: WorkerStatusIdle,
		ctl:    make(ctl, 1),
		proc:   config.DequeueHandler,
		config: config,
	}
	return w
}

func (w *worker) init() {
	w.ctl <- signalInit
}

func (w *worker) sleep() {
	w.ctl <- signalSleep
}

func (w *worker) wakeup() {
	w.ctl <- signalWakeup
}

func (w *worker) stop(force bool) {
	sig := signalStop
	if force {
		sig = signalForceStop
	}
	w.ctl <- sig
}

func (w *worker) dequeue(stream stream) {
	for {
		select {
		case cmd := <-w.ctl:
			w.lastTS = time.Now()
			switch cmd {
			case signalInit:
				if w.c().Verbose(VerboseInfo) {
					w.l().Printf("worker #%d init\n", w.idx)
				}
				w.setStatus(WorkerStatusActive)
				w.m().WorkerInit(w.idx)
			case signalSleep:
				if w.c().Verbose(VerboseInfo) {
					w.l().Printf("worker #%d sleep\n", w.idx)
				}
				w.setStatus(WorkerStatusSleep)
				w.m().WorkerSleep(w.idx)
			case signalWakeup:
				if w.c().Verbose(VerboseInfo) {
					w.l().Printf("worker #%d wakeup\n", w.idx)
				}
				w.setStatus(WorkerStatusActive)
				w.m().WorkerWakeup(w.idx)
			case signalStop, signalForceStop:
				if w.c().Verbose(VerboseInfo) {
					w.l().Printf("worker #%d stop\n", w.idx)
				}
				w.m().WorkerStop(w.idx, cmd == signalForceStop, w.getStatus())
				w.setStatus(WorkerStatusIdle)
				return
			}
		default:
			if w.status == WorkerStatusActive {
				if x, ok := <-stream; ok {
					_ = w.proc.Dequeue(x)
					w.m().QueuePull()
				}
			}
		}
	}
}

func (w *worker) setStatus(status WorkerStatus) {
	atomic.StoreUint32((*uint32)(&w.status), uint32(status))
}

func (w *worker) getStatus() WorkerStatus {
	return WorkerStatus(atomic.LoadUint32((*uint32)(&w.status)))
}

func (w *worker) c() *Config {
	return w.config
}

func (w *worker) m() MetricsWriter {
	return w.config.MetricsHandler
}

func (w *worker) l() Logger {
	return w.config.Logger
}
