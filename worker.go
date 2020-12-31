package queue

type ctl chan signal

type worker struct {
	idx     uint32
	status  status
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
			case signalSleep:
				w.status = wstatusSleep
				w.metrics.WorkerSleep(w.idx)
			case signalInit, signalResume:
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
