package queue

type MetricsWriter interface {
	WorkerSleep(idx uint32)
	WorkerWakeup(idx uint32)
	WorkerStop(idx uint32)
	QueuePut()
	QueuePull()
	QueueLeak()
}
