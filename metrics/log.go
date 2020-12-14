package metrics

import "log"

type Log struct{}

func (m *Log) WorkerSleep(idx uint32) {
	log.Printf("worker %d caught sleep signal\n", idx)
}

func (m *Log) WorkerWakeup(idx uint32) {
	log.Printf("worker %d caught sleep signal\n", idx)
}

func (m *Log) WorkerStop(idx uint32) {
	log.Printf("worker %d caught stop signal\n", idx)
}

func (m *Log) QueuePut() {
	log.Println("new item come to the queue")
}

func (m *Log) QueuePull() {
	log.Println("item leave the queue")
}
