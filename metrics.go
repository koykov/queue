package blqueue

type MetricsWriter interface {
	WorkerSetup(queue string, active, sleep, stop uint)
	WorkerInit(queue string, idx uint32)
	WorkerSleep(queue string, idx uint32)
	WorkerWakeup(queue string, idx uint32)
	WorkerStop(queue string, idx uint32, force bool, status WorkerStatus)
	QueuePut(queue string)
	QueuePull(queue string)
	QueueLeak(queue string)
}
