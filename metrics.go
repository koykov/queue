package blqueue

type MetricsWriter interface {
	WorkerSetup(active, sleep, stop uint)
	WorkerInit(idx uint32)
	WorkerSleep(idx uint32)
	WorkerWakeup(idx uint32)
	WorkerStop(idx uint32)
	WorkerForceStop(idx uint32)
	QueuePut()
	QueuePull()
	QueueLeak()
	QueueClose()
}
