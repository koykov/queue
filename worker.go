package queue

type ctl chan signal

type worker struct {
	status status
	proc   Proc
}

func (w *worker) observe(stream stream, ctl ctl) {
	for {
		select {
		case cmd := <-ctl:
			switch cmd {
			case signalStop:
				w.status = wstatusIdle
				return
			case signalSleep:
				w.status = wstatusSleep
			case signalInit, signalResume:
				w.status = wstatusActive
			}
		default:
			if w.status == wstatusActive {
				w.proc(<-stream)
			}
		}
	}
}
