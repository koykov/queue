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
	pl  uint64
	rri uint64
}

func (e *pq) init(config *Config) error {
	if config.QoS == nil {
		return ErrNoQoS
	}
	e.conf = config
	qos := e.qos()
	e.cp = qos.SummingCapacity()
	e.pl = uint64(len(qos.Queues))
	e.rri = math.MaxUint64

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
	pp := e.qos().Evaluator.Eval(itm.payload)
	if pp == 0 {
		pp = 1
	}
	if pp > 100 {
		pp = 100
	}
	qi := atomic.LoadUint32(&e.prior[pp-1])
	q := e.pool[qi]
	qn := e.qos().Queues[qi].Name
	if !block {
		select {
		case q <- *itm:
			e.mw().SubQueuePut(qn)
			return true
		default:
			e.mw().SubQueueDrop(qn)
			return false
		}
	} else {
		q <- *itm
	}

	e.mw().SubQueuePut(qn)
	return true
}

func (e *pq) getc() chan item {
	return e.egress
}

func (e *pq) pull() item {
	itm := <-e.egress
	e.mw().SubQueuePull(egress)
	return itm
}

func (e *pq) pullOK() (item, bool) {
	itm, ok := <-e.egress
	if ok {
		e.mw().SubQueuePull(egress)
	}
	return itm, ok
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

func (e *pq) close(_ bool) error {
	e.cancel()
	for i := 0; i < len(e.pool); i++ {
		close(e.pool[i])
	}
	close(e.egress)
	return nil
}

func (e *pq) rebalancePB() {
	mxu32 := func(a, b uint32) uint32 {
		if a > b {
			return a
		}
		return b
	}
	qos := e.qos()
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
			qn := e.qos().Queues[i].Name
			e.mw().SubQueuePull(qn)
			e.mw().SubQueuePut(egress)
			e.egress <- itm
			return
		}
	}
}

func (e *pq) shiftRR() {
	pi := atomic.AddUint64(&e.rri, 1) % e.pl
	itm, ok := <-e.pool[pi]
	if ok {
		qn := e.qos().Queues[pi].Name
		e.mw().SubQueuePull(qn)
		e.mw().SubQueuePut(egress)
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

func (e *pq) qos() *QoS {
	return e.conf.QoS
}

func (e *pq) mw() MetricsWriter {
	return e.conf.MetricsWriter
}
