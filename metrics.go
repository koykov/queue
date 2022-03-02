package blqueue

// MetricsWriter is an interface of metrics handler.
// See example of implementations https://github.com/koykov/metrics_writers/tree/master/blqueue.
type MetricsWriter interface {
	// WorkerSetup set initial workers statuses.
	// Calls twice: on queue init and schedule's time range changes.
	WorkerSetup(queue string, active, sleep, stop uint)
	// WorkerInit registers worker's start moment.
	WorkerInit(queue string, idx uint32)
	// WorkerSleep registers when worker puts to sleep.
	WorkerSleep(queue string, idx uint32)
	// WorkerWakeup registers when slept worker resumes.
	WorkerWakeup(queue string, idx uint32)
	// WorkerStop registers when sleeping worker stops.
	WorkerStop(queue string, idx uint32, force bool, status WorkerStatus)
	// QueuePut registers income of new item to the queue.
	QueuePut(queue string)
	// QueuePull registers outgoing of item from the queue.
	QueuePull(queue string)
	// QueueRetry registers total amount of retries.
	QueueRetry(queue string)
	// QueueLeak registers item's leak from the full queue.
	QueueLeak(queue string)
	// QueueLost registers lost items missed queue and DLQ.
	QueueLost(queue string)
}
