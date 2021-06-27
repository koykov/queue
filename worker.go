package blqueue

import (
	"log"
	"sync/atomic"
	"time"
)

type wstatus uint32
type signal uint32

const (
	wstatusIdle wstatus = iota
	wstatusActive
	wstatusSleep

	signalInit signal = iota
	signalSleep
	signalWakeup
	signalStop
	signalForceStop
)

type ctl chan signal

type worker struct {
	idx     uint32
	status  wstatus
	ctl     ctl
	lastTS  time.Time
	proc    Proc
	metrics MetricsWriter
}

func makeWorker(idx uint32, proc Proc, metrics MetricsWriter) *worker {
	w := &worker{
		idx:     idx,
		status:  wstatusIdle,
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
				w.setStatus(wstatusActive)
				w.metrics.WorkerInit(w.idx)
			case signalSleep:
				log.Printf("sleep #%d\n", w.idx)
				w.setStatus(wstatusSleep)
				w.metrics.WorkerSleep(w.idx)
			case signalWakeup:
				log.Printf("resume #%d\n", w.idx)
				w.setStatus(wstatusActive)
				w.metrics.WorkerWakeup(w.idx)
			case signalStop, signalForceStop:
				log.Printf("stop #%d\n", w.idx)
				w.setStatus(wstatusIdle)
				if cmd == signalForceStop {
					w.metrics.WorkerForceStop(w.idx)
				} else {
					w.metrics.WorkerStop(w.idx)
				}
				return
			}
		default:
			if w.status == wstatusActive {
				w.proc(<-stream)
				w.metrics.QueuePull()
			}
		}
	}
}

func (w *worker) setStatus(status wstatus) {
	atomic.StoreUint32((*uint32)(&w.status), uint32(status))
}

func (w *worker) getStatus() wstatus {
	return wstatus(atomic.LoadUint32((*uint32)(&w.status)))
}
