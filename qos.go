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
	if len(q.Queues) == 0 {
		return ErrQoSNoQueues
	}
	for i := 0; i < len(q.Queues); i++ {
		q1 := q.Queues[i]
		if len(q1.Name) == 0 {
			return fmt.Errorf("QoS: queue at index %d has no name", i)
		}
		if q1.Capacity == 0 {
			return fmt.Errorf("QoS: queue #%s has no capacity", q1.Name)
		}
		if q1.Weight == 0 {
			return fmt.Errorf("QoS: queue #%s is senseless due to no weight", q1.Name)
		}
	}
	for i := 0; i < len(q.Queues); i++ {
		for j := 0; j < len(q.Queues); j++ {
			if i == j {
				continue
			}
			if q.Queues[i].Weight == q.Queues[j].Weight {
				return fmt.Errorf("QoS: queues #%s and #%s have the same weights", q.Queues[i].Name, q.Queues[j].Name)
			}
		}
	}
	return nil
}

func (q *QoS) SummingCapacity() (c uint64) {
	if q.EgressCapacity == 0 {
		q.EgressCapacity = defaultEgressCapacity
	}
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

func (q *QoS) Len() int {
	return len(q.Queues)
}

func (q *QoS) Less(i, j int) bool {
	return q.Queues[i].Weight < q.Queues[j].Weight
}

func (q *QoS) Swap(i, j int) {
	q.Queues[i], q.Queues[j] = q.Queues[j], q.Queues[i]
}
