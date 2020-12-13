package queue

type MetricsWriter interface {
	WorkerSleep(uint32 int)
	WorkerWakeup(uint32 int)
	QueuePut()
	QueuePull()
}
