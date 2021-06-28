package blqueue

import (
	"log"
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
	idx     uint32
	status  WorkerStatus
	ctl     ctl
	lastTS  time.Time
	proc    Proc
	metrics MetricsWriter
}

func makeWorker(idx uint32, proc Proc, metrics MetricsWriter) *worker {
	w := &worker{
		idx:     idx,
		status:  WorkerStatusIdle,
		ctl:     make(ctl, 1),
		proc:    proc,
		metrics: metrics,
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
				log.Printf("init #%d\n", w.idx)
				w.setStatus(WorkerStatusActive)
				w.metrics.WorkerInit(w.idx)
			case signalSleep:
				log.Printf("sleep #%d\n", w.idx)
				w.setStatus(WorkerStatusSleep)
				w.metrics.WorkerSleep(w.idx)
			case signalWakeup:
				log.Printf("resume #%d\n", w.idx)
				w.setStatus(WorkerStatusActive)
				w.metrics.WorkerWakeup(w.idx)
			case signalStop, signalForceStop:
				log.Printf("stop #%d\n", w.idx)
				w.metrics.WorkerStop(w.idx, cmd == signalForceStop, w.getStatus())
				w.setStatus(WorkerStatusIdle)
				return
			}
		default:
			if w.status == WorkerStatusActive {
				w.proc(<-stream)
				w.metrics.QueuePull()
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
