package queue

import "log"

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
	proc    Proc
	metrics MetricsWriter
}

func (w *worker) observe(stream stream, ctl ctl) {
	for {
		select {
		case cmd := <-ctl:
			switch cmd {
			case signalStop:
				w.status = wstatusIdle
				w.metrics.WorkerStop(w.idx)
				return
			case signalSleep:
				w.status = wstatusSleep
				w.metrics.WorkerSleep(w.idx)
			case signalInit, signalResume:
				log.Printf("caught init/resume #%d\n", w.idx)
				w.status = wstatusActive
				w.metrics.WorkerWakeup(w.idx)
			}
		default:
			if w.status == wstatusActive {
				w.proc(<-stream)
				w.metrics.QueuePull()
			}
		}
	}
}
