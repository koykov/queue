package queue

type status uint
type signal uint

const (
	qstatusNil      status = 0
	qstatusActive          = 1
	qstatusThrottle        = 2

	wstatusIdle   status = 0
	wstatusActive        = 1
	wstatusSleep         = 2

	signalInit   signal = 0
	signalSleep         = 1
	signalResume        = 2
	signalStop          = 3
)

type Queuer interface {
	Put(x interface{}) bool
}

type Proc func(x interface{})

type queue struct {
	status status
}

type stream chan interface{}
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
