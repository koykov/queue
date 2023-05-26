package qos

import (
	"fmt"
	"strconv"
)

// Algo represent QoS scheduling algorithm.
type Algo uint8

const (
	PQ  Algo = iota // Priority Queuing
	RR              // Round-Robin
	WRR             // Weighted Round-Robin
	// DWRR                // Dynamic Weighted Round-Robin (todo)
	// FQ                  // Fair Queuing (idea?)
	// WFQ                 // Weighted Fair Queuing (idea?)

	Ingress = "ingress"
	Egress  = "egress"
)
const (
	defaultEgressCapacity = uint64(64)
	defaultEgressWorkers  = uint32(1)
)

type Config struct {
	// Chosen algorithm [PQ, RR, WRR].
	Algo Algo
	// Egress sub-queue capacity.
	// If this param omit defaultEgressCapacity (64) will use instead.
	EgressCapacity uint64
	// Count of transit workers between sub-queues and egress sud-queue.
	// If this param omit defaultEgressWorkers (1) will use instead.
	// Use with caution!
	EgressWorkers uint32
	// Helper to determine priority of incoming items.
	// Mandatory param.
	Evaluator PriorityEvaluator
	// Sub-queues config.
	// Mandatory param.
	Queues []Queue
}

// New makes new QoS config using given params.
func New(algo Algo, eval PriorityEvaluator) *Config {
	q := Config{
		Algo:           algo,
		EgressCapacity: defaultEgressCapacity,
		Evaluator:      eval,
	}
	return &q
}

func (q *Config) SetAlgo(algo Algo) *Config {
	q.Algo = algo
	return q
}

func (q *Config) SetEvaluator(eval PriorityEvaluator) *Config {
	q.Evaluator = eval
	return q
}

func (q *Config) SetEgressCapacity(cap uint64) *Config {
	q.EgressCapacity = cap
	return q
}

func (q *Config) SetEgressWorkers(workers uint32) *Config {
	q.EgressWorkers = workers
	return q
}

func (q *Config) AddQueue(subq Queue) *Config {
	if len(subq.Name) == 0 {
		subq.Name = strconv.Itoa(len(q.Queues))
	}
	q.Queues = append(q.Queues, subq)
	return q
}

// Validate check QoS config and returns any error encountered.
func (q *Config) Validate() error {
	if q.Algo > WRR {
		return ErrQoSUnknownAlgo
	}
	if q.Evaluator == nil {
		return ErrQoSNoEvaluator
	}
	if q.EgressCapacity == 0 {
		q.EgressCapacity = defaultEgressCapacity
	}
	if q.EgressWorkers == 0 {
		q.EgressWorkers = defaultEgressWorkers
	}
	if len(q.Queues) == 0 {
		return ErrQoSNoQueues
	}
	if len(q.Queues) == 1 {
		return ErrQoSSenseless
	}
	for i := 0; i < len(q.Queues); i++ {
		q1 := &q.Queues[i]
		if len(q1.Name) == 0 {
			return fmt.Errorf("QoS: queue at index %d has no name", i)
		}
		if q1.Name == Ingress {
			return ErrQoSIngressReserved
		}
		if q1.Name == Egress {
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

// SummingCapacity returns sum of capacities of all sub-queues (including egress).
func (q *Config) SummingCapacity() (c uint64) {
	c += q.EgressCapacity
	for i := 0; i < len(q.Queues); i++ {
		c += q.Queues[i].Capacity
	}
	return
}

// Copy copies config instance to protect queue from changing params after start.
func (q *Config) Copy() *Config {
	cpy := Config{}
	cpy = *q
	cpy.Queues = append([]Queue(nil), q.Queues...)
	return &cpy
}
