package queue

import (
	"fmt"
	"strconv"
)

type QoSAlgo uint8

const (
	PQ  QoSAlgo = iota // Priority Queuing
	RR                 // Round-Robin
	WRR                // Weighted Round-Robin
	// DWRR                // Dynamic Weighted Round-Robin
	// FQ                  // Fair Queuing
	// WFQ                 // Weighted Fair Queuing

	egress = "egress"
)
const defaultEgressCapacity = uint64(64)

type QoSQueue struct {
	Name     string
	Capacity uint64
	Weight   uint64
}

type QoS struct {
	Algo           QoSAlgo
	EgressCapacity uint64
	Evaluator      PriorityEvaluator
	Queues         []QoSQueue
}

func NewQoS(algo QoSAlgo, eval PriorityEvaluator) *QoS {
	q := QoS{
		Algo:           algo,
		EgressCapacity: defaultEgressCapacity,
		Evaluator:      eval,
	}
	return &q
}

func (q *QoS) SetAlgo(algo QoSAlgo) *QoS {
	q.Algo = algo
	return q
}

func (q *QoS) SetEvaluator(eval PriorityEvaluator) *QoS {
	q.Evaluator = eval
	return q
}

func (q *QoS) SetEgressCapacity(cap uint64) *QoS {
	q.EgressCapacity = cap
	return q
}

func (q *QoS) AddQueue(capacity, weight uint64) *QoS {
	name := strconv.Itoa(len(q.Queues))
	return q.AddNamedQueue(name, capacity, weight)
}

func (q *QoS) AddNamedQueue(name string, capacity, weight uint64) *QoS {
	q.Queues = append(q.Queues, QoSQueue{
		Name:     name,
		Capacity: capacity,
		Weight:   weight,
	})
	return q
}

func (q *QoS) Validate() error {
	if q.Algo > WRR {
		return ErrQoSUnknownAlgo
	}
	if q.Evaluator == nil {
		return ErrQoSNoEvaluator
	}
	if q.EgressCapacity == 0 {
		q.EgressCapacity = defaultEgressCapacity
	}
	if len(q.Queues) == 0 {
		return ErrQoSNoQueues
	}
	if len(q.Queues) == 1 {
		return ErrQoSSenseless
	}
	for i := 0; i < len(q.Queues); i++ {
		q1 := q.Queues[i]
		if len(q1.Name) == 0 {
			return fmt.Errorf("QoS: queue at index %d has no name", i)
		}
		if q1.Name == egress {
			return ErrQoSEgressReserved
		}
		if q1.Capacity == 0 {
			return fmt.Errorf("QoS: queue #%s has no capacity", q1.Name)
		}
		if q1.Weight == 0 {
			return fmt.Errorf("QoS: queue #%s is senseless due to no weight", q1.Name)
		}
	}
	return nil
}

func (q *QoS) SummingCapacity() (c uint64) {
	c += q.EgressCapacity
	for i := 0; i < len(q.Queues); i++ {
		c += q.Queues[i].Capacity
	}
	return
}

func (q *QoS) Copy() *QoS {
	cpy := QoS{}
	cpy = *q
	cpy.Queues = append([]QoSQueue(nil), q.Queues...)
	return &cpy
}
