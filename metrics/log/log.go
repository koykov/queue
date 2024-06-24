package queue

import (
	"log"
	"time"

	q "github.com/koykov/queue"
)

// LogMetrics is Log implementation of queue.MetricsWriter.
//
// Don't use in production. Only for debug purposes.
type LogMetrics struct {
	name string
}

var _ = NewLogMetrics

func NewLogMetrics(name string) *LogMetrics {
	m := &LogMetrics{name}
	return m
}

func (m LogMetrics) WorkerSetup(active, sleep, stop uint) {
	log.Printf("queue #%s: setup workers %d active, %d sleep and %d stop", m.name, active, sleep, stop)
}

func (m LogMetrics) WorkerInit(idx uint32) {
	log.Printf("queue %s: worker %d caught init signal\n", m.name, idx)
}

func (m LogMetrics) WorkerSleep(idx uint32) {
	log.Printf("queue %s: worker %d caught sleep signal\n", m.name, idx)
}

func (m LogMetrics) WorkerWakeup(idx uint32) {
	log.Printf("queue %s: worker %d caught wakeup signal\n", m.name, idx)
}

func (m LogMetrics) WorkerWait(idx uint32, delay time.Duration) {
	log.Printf("queue %s: worker %d waits %s\n", m.name, idx, delay)
}

func (m LogMetrics) WorkerStop(idx uint32, force bool, status q.WorkerStatus) {
	if force {
		log.Printf("queue %s: worker %d caught force stop signal (current status %d)\n", m.name, idx, status)
	} else {
		log.Printf("queue %s: worker %d caught stop signal\n", m.name, idx)
	}
}

func (m LogMetrics) QueuePut() {
	log.Printf("queue %s: new item come to the queue\n", m.name)
}

func (m LogMetrics) QueuePull() {
	log.Printf("queue %s: item leave the queue\n", m.name)
}

func (m LogMetrics) QueueRetry() {
	log.Printf("queue %s: retry item processing due to fail\n", m.name)
}

func (m LogMetrics) QueueLeak(dir q.LeakDirection) {
	dirs := "rear"
	if dir == q.LeakDirectionFront {
		dirs = "front"
	}
	log.Printf("queue %s: queue leak from %s\n", m.name, dirs)
}

func (m LogMetrics) QueueDeadline() {
	log.Printf("queue %s: queue deadline\n", m.name)
}

func (m LogMetrics) QueueLost() {
	log.Printf("queue %s: queue lost\n", m.name)
}

func (m LogMetrics) SubQueuePut(subq string) {
	log.Printf("queue %s/%s: new item come to the queue\n", m.name, subq)
}

func (m LogMetrics) SubQueuePull(subq string) {
	log.Printf("queue %s/%s: item leave the queue\n", m.name, subq)
}

func (m LogMetrics) SubQueueDrop(subq string) {
	log.Printf("queue %s/%s: queue drop item\n", m.name, subq)
}
