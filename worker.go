package queue

import (
	"log"
	"time"
)

type wstatus uint
type signal uint

const (
	wstatusIdle wstatus = iota
	wstatusActive
	wstatusSleep

	signalInit signal = iota
	signalSleep
	signalResume
	signalStop
)

type ctl chan signal

type worker struct {
	idx     uint32
	status  wstatus
	lastTS  time.Time
	proc    Proc
	metrics MetricsWriter
}

func (w *worker) observe(stream stream, ctl ctl) {
	for {
		select {
		case cmd := <-ctl:
			w.lastTS = time.Now()
			switch cmd {
			case signalInit:
				log.Printf("init #%d\n", w.idx)
				w.status = wstatusActive
				w.metrics.WorkerInit(w.idx)
			case signalSleep:
				log.Printf("sleep #%d\n", w.idx)
				w.status = wstatusSleep
				w.metrics.WorkerSleep(w.idx)
			case signalResume:
				log.Printf("resume #%d\n", w.idx)
				w.status = wstatusActive
				w.metrics.WorkerWakeup(w.idx)
			case signalStop:
				log.Printf("stop #%d\n", w.idx)
				w.status = wstatusIdle
				w.metrics.WorkerStop(w.idx)
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
