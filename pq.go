package queue

import (
	"context"
	"math"
	"sync/atomic"
)

type pq struct {
	pool   []chan item
	egress chan item
	prior  [100]uint32
	conf   *Config
	cancel context.CancelFunc

	cp  uint64
	rri uint64
}

func (e *pq) init(config *Config) error {
	if config.QoS == nil {
		return ErrNoQoS
	}
	e.conf = config
	qos := e.conf.QoS
	e.cp = qos.SummingCapacity()

	// Priorities buffer calculation.
	e.rebalancePB()

	// Create channels.
	for i := 0; i < len(qos.Queues); i++ {
		e.pool = append(e.pool, make(chan item, qos.Queues[i].Capacity))
	}
	e.egress = make(chan item, qos.EgressCapacity)

	// Start scheduler.
	var ctx context.Context
	ctx, e.cancel = context.WithCancel(context.Background())
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				switch qos.Algo {
				case PQ:
					e.shiftPQ()
				case RR:
					e.shiftRR()
				case WRR:
					//
				}
			}
		}
	}(ctx)
	return nil
}

func (e *pq) put(itm *item, block bool) bool {
	pp := e.conf.QoS.Evaluator.Eval(itm.payload)
	if pp == 0 {
		pp = 1
	}
	if pp > 100 {
		pp = 100
	}
	qi := e.prior[pp-1]
	q := e.pool[qi]
	if !block {
		select {
		case q <- *itm:
			return true
		default:
			return false
		}
	} else {
		q <- *itm
	}

	return true
}

func (e *pq) getc() chan item {
	return e.egress
}

func (e *pq) size() (sz int) {
	for i := 0; i < len(e.pool); i++ {
		sz += len(e.pool[i])
	}
	sz += len(e.egress)
	return
}

func (e *pq) cap() int {
	return int(e.cp)
}

func (e *pq) close(force bool) error {
	return nil
}

func (e *pq) rebalancePB() {
	mxu32 := func(a, b uint32) uint32 {
		if a > b {
			return a
		}
		return b
	}
	qos := e.conf.QoS
	var tw uint64
	for i := 0; i < len(qos.Queues); i++ {
		tw += atomic.LoadUint64(&qos.Queues[i].Weight)
	}
	var qi uint32
	for i := 0; i < len(qos.Queues); i++ {
		rate := math.Round(float64(atomic.LoadUint64(&qos.Queues[i].Weight)) / float64(tw) * 100)
		mxp := uint32(rate)
		for j := qi; j < mxu32(qi+mxp, 100); j++ {
			atomic.StoreUint32(&e.prior[j], uint32(i))
		}
		qi += mxp
	}
}

func (e *pq) shiftPQ() {
	for i := 0; i < len(e.pool); i++ {
		itm, ok := <-e.pool[i]
		if ok {
			e.egress <- itm
			return
		}
	}
}

func (e *pq) shiftRR() {
	pi := atomic.AddUint64(&e.rri, 1) % uint64(len(e.pool))
	itm, ok := <-e.pool[pi]
	if ok {
		e.egress <- itm
	}
}

func (e *pq) assertPB(expect [100]uint32) (int, bool) {
	for i := 0; i < 100; i++ {
		if e.prior[i] != expect[i] {
			return i, false
		}
	}
	return -1, true
}
