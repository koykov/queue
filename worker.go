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
	proc   DequeueWorker
	config *Config
}

func makeWorker(idx uint32, config *Config) *worker {
	w := &worker{
		idx:    idx,
		status: WorkerStatusIdle,
		ctl:    make(ctl, 1),
		proc:   config.DequeueWorker,
		config: config,
	}
	return w
}

func (w *worker) signal(sig signal) {
	w.ctl <- sig
}

func (w *worker) dequeue(stream stream) {
	for {
		switch w.getStatus() {
		case WorkerStatusSleep:
			cmd := <-w.ctl
			switch cmd {
			case signalStop, signalForceStop:
				w.stop(cmd == signalForceStop)
				return
			case signalWakeup:
				w.wakeup()
			}
		case WorkerStatusActive:
			select {
			case cmd := <-w.ctl:
				w.lastTS = time.Now()
				switch cmd {
				case signalInit:
					w.init()
				case signalSleep:
					w.sleep()
				case signalWakeup:
					w.wakeup()
				case signalStop, signalForceStop:
					w.stop(cmd == signalForceStop)
					return
				}
			case x, ok := <-stream:
				if !ok {
					w.stop(true)
					return
				}
				_ = w.proc.Dequeue(x)
				w.m().QueuePull()
			}
		case WorkerStatusIdle:
			cmd := <-w.ctl
			if cmd == signalInit {
				w.init()
			}
		}
	}
}

func (w *worker) init() {
	if w.c().Verbose(VerboseInfo) {
		w.l().Printf("worker #%d init\n", w.idx)
	}
	w.setStatus(WorkerStatusActive)
	w.m().WorkerInit(w.idx)
}

func (w *worker) sleep() {
	if w.c().Verbose(VerboseInfo) {
		w.l().Printf("worker #%d sleep\n", w.idx)
	}
	w.setStatus(WorkerStatusSleep)
	w.m().WorkerSleep(w.idx)
}

func (w *worker) wakeup() {
	if w.c().Verbose(VerboseInfo) {
		w.l().Printf("worker #%d wakeup\n", w.idx)
	}
	w.setStatus(WorkerStatusActive)
	w.m().WorkerWakeup(w.idx)
}

func (w *worker) stop(force bool) {
	if w.c().Verbose(VerboseInfo) {
		msg := "worker #%d stop\n"
		if force {
			msg = "worker #%d force stop\n"
		}
		w.l().Printf(msg, w.idx)
	}
	w.m().WorkerStop(w.idx, force, w.getStatus())
	w.setStatus(WorkerStatusIdle)
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
	return w.config.MetricsWriter
}

func (w *worker) l() Logger {
	return w.config.Logger
}
