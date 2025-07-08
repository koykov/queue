package log

import (
	"log"
	"time"
)

// MetricsWriter is Log implementation of queue.MetricsWriter.
//
// Don't use in production. Only for debug purposes.
type MetricsWriter struct {
	name string
}

var _ = NewLogMetrics

func NewLogMetrics(name string) *MetricsWriter {
	m := &MetricsWriter{name}
	return m
}

func (w MetricsWriter) WorkerSetup(active, sleep, stop uint) {
	log.Printf("queue #%s: setup workers %d active, %d sleep and %d stop", w.name, active, sleep, stop)
}

func (w MetricsWriter) WorkerInit(idx uint32) {
	log.Printf("queue %s: worker %d caught init signal\n", w.name, idx)
}

func (w MetricsWriter) WorkerSleep(idx uint32) {
	log.Printf("queue %s: worker %d caught sleep signal\n", w.name, idx)
}

func (w MetricsWriter) WorkerWakeup(idx uint32) {
	log.Printf("queue %s: worker %d caught wakeup signal\n", w.name, idx)
}

func (w MetricsWriter) WorkerWait(idx uint32, delay time.Duration) {
	log.Printf("queue %s: worker %d waits %s\n", w.name, idx, delay)
}

func (w MetricsWriter) WorkerStop(idx uint32, force bool, status string) {
	if force {
		log.Printf("queue %s: worker %d caught force stop signal (current status %s)\n", w.name, idx, status)
	} else {
		log.Printf("queue %s: worker %d caught stop signal\n", w.name, idx)
	}
}

func (w MetricsWriter) QueuePut() {
	log.Printf("queue %s: new item come to the queue\n", w.name)
}

func (w MetricsWriter) QueuePull() {
	log.Printf("queue %s: item leave the queue\n", w.name)
}

func (w MetricsWriter) QueueRetry(delay time.Duration) {
	if delay > 0 {
		log.Printf("queue %s: retry item processing due to fail with delay %s\n", w.name, delay)
		return
	}
	log.Printf("queue %s: retry item processing due to fail\n", w.name)
}

func (w MetricsWriter) QueueLeak(direction string) {
	log.Printf("queue %s: queue leak from %s\n", w.name, direction)
}

func (w MetricsWriter) QueueDeadline() {
	log.Printf("queue %s: queue deadline\n", w.name)
}

func (w MetricsWriter) QueueLost() {
	log.Printf("queue %s: queue lost\n", w.name)
}

func (w MetricsWriter) SubqPut(subq string) {
	log.Printf("queue %s/%s: new item come to the queue\n", w.name, subq)
}

func (w MetricsWriter) SubqPull(subq string) {
	log.Printf("queue %s/%s: item leave the queue\n", w.name, subq)
}

func (w MetricsWriter) SubqLeak(subq string) {
	log.Printf("queue %s/%s: queue leak item\n", w.name, subq)
}
