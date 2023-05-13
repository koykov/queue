package queue

type QoSAlgo uint8

const (
	PQ  QoSAlgo = iota // Priority Queuing
	RR                 // Round-Robin
	WRR                // Weighted Round-Robin
	FQ                 // Fair Queuing
	WFQ                // Weighted Fair Queuing (in fact FWFQ - Flow-based WFQ)
)

type QoS struct {
	Algo      QoSAlgo
	Egress    uint64
	Evaluator PriorityEvaluator
	Queues    []QoSQueue
}

type QoSQueue struct {
	Name            string
	CapacityPercent uint8
}

type PriorityEvaluator interface {
	Eval(float64) uint
}
