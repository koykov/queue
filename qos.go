package queue

const (
	defaultEgressPercent = uint8(5)
	defaultRRPacketSize  = uint16(4)
)

type QoSAlgo uint8

const (
	PQ  QoSAlgo = iota // Priority Queuing
	RR                 // Round-Robin
	WRR                // Weighted Round-Robin
	FQ                 // Fair Queuing
	WFQ                // Weighted Fair Queuing (in fact FWFQ - Flow-based WFQ)
)

type QoS struct {
	Algo          QoSAlgo
	EgressPercent uint8
	RRPacketSize  uint16
	Evaluator     PriorityEvaluator
	Queues        []QoSQueue
}

type QoSQueue struct {
	Name            string
	CapacityPercent uint8
}
