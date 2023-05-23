package queue

import (
	"context"
	"math"
	"sync/atomic"
)

type pq struct {
	pool    []chan item
	egress  chan item
	inprior [100]uint32
	eprior  [100]uint32
	conf    *Config
	cancel  context.CancelFunc

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
					e.shiftWRR()
				}
			}
		}
	}(ctx)
	return nil
}

func (e *pq) enqueue(itm *item, block bool) bool {
	pp := e.qos().Evaluator.Eval(itm.payload)
	if pp == 0 {
		pp = 1
	}
	if pp > 100 {
		pp = 100
	}
	itm.subqi = atomic.LoadUint32(&e.inprior[pp-1])
	q := e.pool[itm.subqi]
	qn := e.qos().Queues[itm.subqi].Name
	e.mw().SubQueuePut(qn)
	if !block {
		select {
		case q <- *itm:
			return true
		default:
			e.mw().SubQueueDrop(qn)
			return false
		}
	} else {
		q <- *itm
	}

	return true
}

func (e *pq) dequeue(block bool) (item, bool) {
	if block {
		itm := <-e.egress
		e.mw().SubQueuePull(egress)
		return itm, true
	}
	itm, ok := <-e.egress
	if ok {
		e.mw().SubQueuePull(egress)
	}
	return itm, ok
}

func (e *pq) dequeueSQ(subqi uint32, block bool) (item, bool) {
	qn := e.qos().Queues[subqi].Name
	if block {
		itm := <-e.pool[subqi]
		e.mw().SubQueuePull(qn)
		return itm, true
	}
	itm, ok := <-e.pool[subqi]
	if ok {
		e.mw().SubQueuePull(qn)
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
	lim := func(x, lim uint32) uint32 {
		if x > lim {
			return lim
		}
		return x
	}
	qos := e.qos()
	var tw uint64
	for i := 0; i < len(qos.Queues); i++ {
		tw += atomic.LoadUint64(&qos.Queues[i].IngressWeight)
	}
	var qi uint32
	for i := 0; i < len(qos.Queues); i++ {
		rate := math.Ceil(float64(atomic.LoadUint64(&qos.Queues[i].IngressWeight)) / float64(tw) * 100)
		mxp := uint32(rate)
		for j := qi; j < mxu32(qi+mxp, 100); j++ {
			atomic.StoreUint32(&e.inprior[lim(j, 99)], uint32(i))
		}
		qi += mxp
	}

	tw, qi = 0, 0
	for i := 0; i < len(qos.Queues); i++ {
		tw += atomic.LoadUint64(&qos.Queues[i].EgressWeight)
	}
	for i := 0; i < len(qos.Queues); i++ {
		rate := math.Ceil(float64(atomic.LoadUint64(&qos.Queues[i].EgressWeight)) / float64(tw) * 100)
		mxp := uint32(rate)
		for j := qi; j < mxu32(qi+mxp, 100); j++ {
			atomic.StoreUint32(&e.eprior[lim(j, 99)], uint32(i))
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
			e.egress <- itm
			e.mw().SubQueuePut(egress)
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

func (e *pq) shiftWRR() {
	// ...
}

func (e *pq) assertPT(expectIPT, expectEPT [100]uint32) (int, bool) {
	for i := 0; i < 100; i++ {
		if e.inprior[i] != expectIPT[i] {
			return i, false
		}
	}
	for i := 0; i < 100; i++ {
		if e.eprior[i] != expectEPT[i] {
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
