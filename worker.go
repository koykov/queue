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

	sigInit signal = iota
	sigSleep
	sigWakeup
	sigStop
	sigForceStop
)

type worker struct {
	idx    uint32
	status WorkerStatus
	pause  chan struct{}
	lastTS int64
	proc   DequeueWorker
	config *Config
}

func makeWorker(idx uint32, config *Config) *worker {
	w := &worker{
		idx:    idx,
		status: WorkerStatusIdle,
		pause:  make(chan struct{}, 1),
		proc:   config.DequeueWorker,
		config: config,
	}
	return w
}

func (w *worker) signal(sig signal) {
	atomic.StoreInt64(&w.lastTS, time.Now().UnixNano())
	switch sig {
	case sigInit:
		w.init()
	case sigSleep:
		w.sleep()
	case sigWakeup:
		w.wakeup()
	case sigStop, sigForceStop:
		w.stop(sig == sigForceStop)
	}
}

func (w *worker) dequeue(stream stream) {
	for {
		switch w.getStatus() {
		case WorkerStatusSleep:
			<-w.pause
		case WorkerStatusActive:
			x, ok := <-stream
			if !ok {
				w.stop(true)
				return
			}
			_ = w.proc.Dequeue(x)
			w.m().QueuePull(w.k())
		case WorkerStatusIdle:
			return
		}
	}
}

func (w *worker) init() {
	if w.l() != nil {
		w.l().Printf("queue #%s worker #%d init\n", w.k(), w.idx)
	}
	w.setStatus(WorkerStatusActive)
	w.m().WorkerInit(w.k(), w.idx)
}

func (w *worker) sleep() {
	if w.l() != nil {
		w.l().Printf("queue #%s worker #%d sleep\n", w.k(), w.idx)
	}
	w.setStatus(WorkerStatusSleep)
	w.m().WorkerSleep(w.k(), w.idx)
}

func (w *worker) wakeup() {
	if w.l() != nil {
		w.l().Printf("queue #%s worker #%d wakeup\n", w.k(), w.idx)
	}
	w.setStatus(WorkerStatusActive)
	w.m().WorkerWakeup(w.k(), w.idx)
	w.pause <- struct{}{}
}

func (w *worker) stop(force bool) {
	if w.l() != nil {
		msg := "queue #%s worker #%d stop\n"
		if force {
			msg = "queue #%s worker #%d force stop\n"
		}
		w.l().Printf(msg, w.k(), w.idx)
	}
	w.m().WorkerStop(w.k(), w.idx, force, w.getStatus())
	w.setStatus(WorkerStatusIdle)
	w.pause <- struct{}{}
}

func (w *worker) setStatus(status WorkerStatus) {
	atomic.StoreUint32((*uint32)(&w.status), uint32(status))
}

func (w *worker) getStatus() WorkerStatus {
	return WorkerStatus(atomic.LoadUint32((*uint32)(&w.status)))
}

func (w *worker) sleptEnough() bool {
	dur := time.Duration(time.Now().UnixNano() - atomic.LoadInt64(&w.lastTS))
	return dur >= w.c().SleepTimeout
}

func (w *worker) c() *Config {
	return w.config
}

func (w *worker) k() string {
	return w.config.Key
}

func (w *worker) m() MetricsWriter {
	return w.config.MetricsWriter
}

func (w *worker) l() Logger {
	return w.config.Logger
}
