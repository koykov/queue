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

// Worker implementation.
type worker struct {
	// Index of worker in the pool.
	// For logging purposes.
	idx uint32
	// Status of the worker.
	status WorkerStatus
	// Pause channel between put to sleep and stop.
	pause chan struct{}
	// Last signal timestamp.
	lastTS int64
	// Dequeuer instance.
	proc Dequeuer
	// Config of the queue.
	config *Config
}

// Make new idle worker.
func makeWorker(idx uint32, config *Config) *worker {
	w := &worker{
		idx:    idx,
		status: WorkerStatusIdle,
		pause:  make(chan struct{}, 1),
		proc:   config.Dequeuer,
		config: config,
	}
	return w
}

// Signal handler.
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

// Stream processing.
func (w *worker) dequeue(stream stream) {
	for {
		switch w.getStatus() {
		case WorkerStatusSleep:
			// Wait config.SleepTimeout.
			<-w.pause
		case WorkerStatusActive:
			// Read item from the stream.
			x, ok := <-stream
			if !ok {
				// Stream is closed. Immediately stop and exit.
				w.stop(true)
				return
			}
			// Forward item to dequeuer.
			_ = w.proc.Dequeue(x)
			w.m().QueuePull(w.k())
		case WorkerStatusIdle:
			// Exit on idle status.
			return
		}
	}
}

// Start idle worker.
func (w *worker) init() {
	if w.l() != nil {
		w.l().Printf("queue #%s worker #%d init\n", w.k(), w.idx)
	}
	w.setStatus(WorkerStatusActive)
	w.m().WorkerInit(w.k(), w.idx)
}

// Put worker to the sleep.
func (w *worker) sleep() {
	if w.l() != nil {
		w.l().Printf("queue #%s worker #%d sleep\n", w.k(), w.idx)
	}
	w.setStatus(WorkerStatusSleep)
	w.m().WorkerSleep(w.k(), w.idx)
}

// Wakeup sleeping worker.
func (w *worker) wakeup() {
	if w.l() != nil {
		w.l().Printf("queue #%s worker #%d wakeup\n", w.k(), w.idx)
	}
	w.setStatus(WorkerStatusActive)
	w.m().WorkerWakeup(w.k(), w.idx)
	w.pause <- struct{}{}
}

// Stop (or force stop) worker.
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
	// Notify pause channel about stop.
	w.pause <- struct{}{}
}

// Set worker status.
func (w *worker) setStatus(status WorkerStatus) {
	atomic.StoreUint32((*uint32)(&w.status), uint32(status))
}

// Get worker status.
func (w *worker) getStatus() WorkerStatus {
	return WorkerStatus(atomic.LoadUint32((*uint32)(&w.status)))
}

// Check if worker slept enough time.
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
