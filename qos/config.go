package qos

import (
	"fmt"
	"strconv"
	"time"
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
	defaultEgressCapacity      = uint64(64)
	defaultEgressWorkers       = uint32(1)
	defaultEgressIdleThreshold = uint32(1000)
	defaultEgressIdleTimeout   = time.Millisecond
)

type Config struct {
	// Chosen algorithm [PQ, RR, WRR].
	Algo Algo
	// Egress sub-queue and workers settings.
	Egress EgressConfig
	// Helper to determine priority of incoming items.
	// Mandatory param.
	Evaluator PriorityEvaluator
	// Sub-queues config.
	// Mandatory param.
	Queues []Queue
}

type EgressConfig struct {
	// Egress sub-queue capacity.
	// If this param omit defaultEgressCapacity (64) will use instead.
	Capacity uint64
	// Count of transit workers between sub-queues and egress sud-queue.
	// If this param omit defaultEgressWorkers (1) will use instead.
	// Use with caution!
	Workers uint32
	// Limit of idle read attempts.
	// If this param omit defaultEgressIdleThreshold (1000) will use instead.
	IdleThreshold uint32
	// Time to wait after IdleThreshold read attempts.
	// If this param omit defaultEgressIdleTimeout (1ms) will use instead.
	IdleTimeout time.Duration
}

// New makes new QoS config using given params.
func New(algo Algo, eval PriorityEvaluator) *Config {
	q := Config{
		Algo:      algo,
		Egress:    EgressConfig{Capacity: defaultEgressCapacity},
		Evaluator: eval,
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
	q.Egress.Capacity = cap
	return q
}

func (q *Config) SetEgressWorkers(workers uint32) *Config {
	q.Egress.Workers = workers
	return q
}

func (q *Config) SetEgressIdleThreshold(threshold uint32) *Config {
	q.Egress.IdleThreshold = threshold
	return q
}

func (q *Config) SetEgressIdleTimeout(timeout time.Duration) *Config {
	q.Egress.IdleTimeout = timeout
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
		return ErrUnknownAlgo
	}
	if q.Evaluator == nil {
		return ErrNoEvaluator
	}
	if q.Egress.Capacity == 0 {
		q.Egress.Capacity = defaultEgressCapacity
	}
	if q.Egress.Workers == 0 {
		q.Egress.Workers = defaultEgressWorkers
	}
	if q.Egress.IdleThreshold == 0 {
		q.Egress.IdleThreshold = defaultEgressIdleThreshold
	}
	if q.Egress.IdleTimeout == 0 {
		q.Egress.IdleTimeout = defaultEgressIdleTimeout
	}
	if len(q.Queues) == 0 {
		return ErrNoQueues
	}
	if len(q.Queues) == 1 {
		return ErrSenseless
	}
	for i := 0; i < len(q.Queues); i++ {
		q1 := &q.Queues[i]
		if len(q1.Name) == 0 {
			return fmt.Errorf("QoS: queue at index %d has no name", i)
		}
		if q1.Name == Ingress || q1.Name == Egress {
			return ErrNameReserved
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
	c += q.Egress.Capacity
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
