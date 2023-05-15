package queue

type QoSAlgo uint8

const (
	PQ  QoSAlgo = iota // Priority Queuing
	RR                 // Round-Robin
	WRR                // Weighted Round-Robin
	// FQ                 // Fair Queuing
	// WFQ                // Weighted Fair Queuing
)

type QoS struct {
	Algo           QoSAlgo
	EgressCapacity uint64
	Evaluator      PriorityEvaluator
	Queues         []QoSQueue
}

type QoSQueue struct {
	Name     string
	Capacity uint64
	Weight   uint64
}
