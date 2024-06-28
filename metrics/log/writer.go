package log

import (
	"log"
	"time"

	q "github.com/koykov/queue"
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

func (w MetricsWriter) WorkerStop(idx uint32, force bool, status q.WorkerStatus) {
	if force {
		log.Printf("queue %s: worker %d caught force stop signal (current status %d)\n", w.name, idx, status)
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

func (w MetricsWriter) QueueRetry() {
	log.Printf("queue %s: retry item processing due to fail\n", w.name)
}

func (w MetricsWriter) QueueLeak(dir q.LeakDirection) {
	dirs := "rear"
	if dir == q.LeakDirectionFront {
		dirs = "front"
	}
	log.Printf("queue %s: queue leak from %s\n", w.name, dirs)
}

func (w MetricsWriter) QueueDeadline() {
	log.Printf("queue %s: queue deadline\n", w.name)
}

func (w MetricsWriter) QueueLost() {
	log.Printf("queue %s: queue lost\n", w.name)
}

func (w MetricsWriter) SubQueuePut(subq string) {
	log.Printf("queue %s/%s: new item come to the queue\n", w.name, subq)
}

func (w MetricsWriter) SubQueuePull(subq string) {
	log.Printf("queue %s/%s: item leave the queue\n", w.name, subq)
}

func (w MetricsWriter) SubQueueDrop(subq string) {
	log.Printf("queue %s/%s: queue drop item\n", w.name, subq)
}
