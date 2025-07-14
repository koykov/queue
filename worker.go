package queue

import (
	"sync/atomic"
	"time"
)

type WorkerStatus uint32

const (
	WorkerStatusIdle WorkerStatus = iota
	WorkerStatusActive
	WorkerStatusSleep
)

func (s WorkerStatus) String() string {
	switch s {
	case WorkerStatusIdle:
		return "idle"
	case WorkerStatusActive:
		return "active"
	case WorkerStatusSleep:
		return "sleep"
	}
	return "unknown"
}

type signal uint32

const (
	sigInit signal = iota
	sigSleep
	sigWakeup
	sigStop
	sigForceStop
)

// Internal worker implementation.
type worker struct {
	// Index of worker in the pool.
	// For logging purposes.
	idx uint32
	// Status of the worker.
	status WorkerStatus
	// Channel to control sleep and stop states.
	// This channel delivers two signals:
	// * wakeup for slept workers
	// * force close for active workers
	ctl chan struct{}
	// Last signal timestamp.
	lastTS int64
	// Worker instance.
	proc Worker
	// Config of the queue.
	config *Config
}

// Make new idle worker.
func makeWorker(idx uint32, config *Config) *worker {
	w := &worker{
		idx:    idx,
		status: WorkerStatusIdle,
		ctl:    make(chan struct{}, 1),
		proc:   config.Worker,
		config: config,
	}
	return w
}

// Send signal to worker.
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

// Waits to income item to process or control signal.
func (w *worker) await(queue *Queue) {
	for {
		switch w.getStatus() {
		case WorkerStatusSleep:
			// Wait config.SleepInterval.
			<-w.ctl
		case WorkerStatusActive:
			// Read itm from the stream.
			itm, ok := queue.engine.dequeue()
			if !ok {
				// Stream is closed. Immediately stop and exit.
				w.stop(true)
				return
			}

			// Check deadline.
			if itm.deadline > 0 {
				now := queue.clk().Now().UnixNano()
				if now-itm.deadline >= 0 {
					if queue.CheckBit(flagLeaky) && w.c().DeadlineToDLQ {
						_ = w.c().DLQ.Enqueue(itm.payload)
					}
					w.mw().QueueDeadline()
					continue
				}
			}

			w.mw().QueuePull()

			var intr bool
			// Check delayed execution.
			if itm.delay > 0 {
				now := queue.clk().Now().UnixNano()
				if delta := time.Duration(itm.delay - now); delta > 0 {
					// Processing time has not yet arrived. So wait till delay ends.
					select {
					case <-time.After(delta):
						break
					case <-w.ctl:
						// Waiting interrupted due to force close signal.
						intr = true
						// Calculate real wait time.
						delta = time.Duration(queue.clk().Now().UnixNano() - now)
						break
					}
					w.mw().WorkerWait(w.idx, delta)
				}
			}
			if intr {
				// Return item back to the queue due to interrupt signal.
				_ = queue.renqueue(&itm)
				return
			}

			// Forward itm to dequeuer.
			if err := w.proc.Do(itm.payload); err != nil {
				// Processing failed.
				if itm.retries < w.c().MaxRetries {
					// Try to retry processing if possible.
					delay := w.c().Backoff.Next(w.c().RetryInterval, int(itm.retries))
					if delay > 0 {
						// Apply jitter logic to precalculated delay.
						delay = w.c().Jitter.Apply(delay)
						select {
						case <-time.After(delay):
							// Wait for interval calculated by Backoff + Jitter.
							break
						case <-w.ctl:
							intr = true
						}
					}
					if !intr {
						w.mw().QueueRetry(delay)
						itm.retries++
						itm.delay = 0 // Clear item timestamp for 2nd, 3rd, ... attempts.
						_ = queue.renqueue(&itm)
					}
				} else if queue.CheckBit(flagLeaky) && w.c().FailToDLQ {
					_ = w.c().DLQ.Enqueue(itm.payload)
					w.mw().QueueLeak(LeakDirectionFront.String())
				}
			}
		case WorkerStatusIdle:
			// Exit on idle status.
			return
		}
	}
}

// Start idle worker.
func (w *worker) init() {
	if w.l() != nil {
		w.l().Printf("worker #%d init\n", w.idx)
	}
	w.setStatus(WorkerStatusActive)
	w.mw().WorkerInit(w.idx)
}

// Put worker to the sleep.
func (w *worker) sleep() {
	if w.l() != nil {
		w.l().Printf("worker #%d sleep\n", w.idx)
	}
	w.setStatus(WorkerStatusSleep)
	w.mw().WorkerSleep(w.idx)
}

// Wakeup sleeping worker.
func (w *worker) wakeup() {
	if w.l() != nil {
		w.l().Printf("worker #%d wakeup\n", w.idx)
	}
	w.setStatus(WorkerStatusActive)
	w.mw().WorkerWakeup(w.idx)
	w.notifyCtl()
}

// Stop (or force stop) worker.
func (w *worker) stop(force bool) {
	if w.l() != nil {
		msg := "worker #%d stop\n"
		if force {
			msg = "worker #%d force stop\n"
		}
		w.l().Printf(msg, w.idx)
	}
	w.mw().WorkerStop(w.idx, force, w.getStatus().String())
	w.setStatus(WorkerStatusIdle)
	w.notifyCtl()
}

// Check if ctl channel is empty and send signal (wakeup or force close).
func (w *worker) notifyCtl() {
	// Check ctl channel for previously undelivered signal.
	if len(w.ctl) > 0 {
		// Clear ctl channel to prevent locking.
		_, _ = <-w.ctl
	}

	// Send stop signal to ctl channel.
	w.ctl <- struct{}{}
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
	dur := time.Duration(w.c().Clock.Now().UnixNano() - atomic.LoadInt64(&w.lastTS))
	return dur >= w.c().SleepInterval
}

func (w *worker) c() *Config {
	return w.config
}

func (w *worker) mw() MetricsWriter {
	return w.config.MetricsWriter
}

func (w *worker) l() Logger {
	return w.config.Logger
}
