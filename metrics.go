package queue

import "time"

// MetricsWriter is an interface of queue metrics handler.
// See example of implementations https://github.com/koykov/metrics_writers/tree/master/queue.
type MetricsWriter interface {
	// WorkerSetup set initial workers statuses.
	// Calls twice: on queue init and schedule's time range changes.
	WorkerSetup(active, sleep, stop uint)
	// WorkerInit registers worker's start moment.
	WorkerInit(idx uint32)
	// WorkerSleep registers when worker puts to sleep.
	WorkerSleep(idx uint32)
	// WorkerWakeup registers when slept worker resumes.
	WorkerWakeup(idx uint32)
	// WorkerWait registers how many worker waits due to delayed execution.
	WorkerWait(idx uint32, dur time.Duration)
	// WorkerStop registers when sleeping worker stops.
	WorkerStop(idx uint32, force bool, status string)
	// QueuePut registers income of new item to the queue.
	QueuePut()
	// QueuePull registers outgoing of item from the queue.
	QueuePull()
	// QueueRetry registers total amount of retries.
	QueueRetry()
	// QueueLeak registers item's leak from the full queue.
	// Param dir indicates leak direction and may be "rear" or "front".
	QueueLeak(direction string)
	// QueueDeadline registers amount of skipped processing of items due to deadline.
	QueueDeadline()
	// QueueLost registers lost items missed queue and DLQ.
	QueueLost()

	// SubqPut registers income of new item to the sub-queue.
	SubqPut(subq string)
	// SubqPull registers outgoing of item from the sub-queue.
	SubqPull(subq string)
	// SubqLeak registers item's drop from the full queue.
	SubqLeak(subq string)
}
