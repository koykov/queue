package queue

import (
	"context"
	"math"
	"sync/atomic"

	"github.com/koykov/queue/qos"
)

type pq struct {
	pool    []chan item
	egress  chan item
	inprior [100]uint32
	eprior  [100]uint32
	conf    *Config
	cancel  context.CancelFunc

	ew   int32
	cp   uint64
	ql   uint64
	rri  uint64
	wrri uint64
}

func (e *pq) init(config *Config) error {
	if config.QoS == nil {
		return qos.ErrNoQoS
	}
	e.conf = config
	q := e.qos()
	e.cp = q.SummingCapacity()
	e.ql = uint64(len(q.Queues))
	e.rri, e.wrri = math.MaxUint64, math.MaxUint64

	// Priorities buffer calculation.
	e.rebalancePB()

	// Create channels.
	for i := 0; i < len(q.Queues); i++ {
		e.pool = append(e.pool, make(chan item, q.Queues[i].Capacity))
	}
	e.egress = make(chan item, q.EgressCapacity)

	// Start scheduler.
	var ctx context.Context
	ctx, e.cancel = context.WithCancel(context.Background())
	for i := uint32(0); i < q.EgressWorkers; i++ {
		atomic.AddInt32(&e.ew, 1)
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					atomic.AddInt32(&e.ew, -1)
					return
				default:
					switch q.Algo {
					case qos.PQ:
						e.shiftPQ()
					case qos.RR:
						e.shiftRR()
					case qos.WRR:
						e.shiftWRR()
					}
				}
			}
		}(ctx)
	}
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
	qn := e.qn(itm.subqi)
	e.mw().SubqPut(qn)
	if !block {
		select {
		case q <- *itm:
			return true
		default:
			e.mw().SubqLeak(qn)
			return false
		}
	} else {
		q <- *itm
	}

	return true
}

func (e *pq) dequeue() (item, bool) {
	itm, ok := <-e.egress
	if ok {
		e.mw().SubqPull(qos.Egress)
	}
	return itm, ok
}

func (e *pq) dequeueSQ(subqi uint32) (item, bool) {
	itm, ok := <-e.pool[subqi]
	if ok {
		e.mw().SubqPull(e.qn(subqi))
	}
	return itm, ok
}

func (e *pq) size() (sz int) {
	return e.size1(true)
}

func (e *pq) size1(includingEgress bool) (sz int) {
	for i := 0; i < len(e.pool); i++ {
		sz += len(e.pool[i])
	}
	if includingEgress {
		sz += len(e.egress)
	}
	return
}

func (e *pq) cap() int {
	return int(e.cp)
}

func (e *pq) close(_ bool) error {
	// Spinlock waiting till sub-queues isn't empty.
	for e.size1(false) > 0 {
	}
	// Stop egress workers.
	e.cancel()
	// Close sub-queues channels.
	for i := 0; i < len(e.pool); i++ {
		close(e.pool[i])
	}
	// Spinlock waiting till all egress workers finished.
	for atomic.LoadInt32(&e.ew) > 0 {
	}
	// Close egress channel.
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
	q := e.qos()
	// Build ingress priority table.
	var tw uint64
	for i := 0; i < len(q.Queues); i++ {
		tw += atomic.LoadUint64(&q.Queues[i].Weight)
	}
	var qi uint32
	for i := 0; i < len(q.Queues); i++ {
		rate := math.Ceil(float64(atomic.LoadUint64(&q.Queues[i].Weight)) / float64(tw) * 100)
		mxp := uint32(rate)
		for j := qi; j < mxu32(qi+mxp, 100); j++ {
			atomic.StoreUint32(&e.inprior[lim(j, 99)], uint32(i))
		}
		qi += mxp
	}

	// Build and shuffle egress priority table.
	var mnw uint64 = math.MaxUint64
	for i := 0; i < len(q.Queues); i++ {
		ew := atomic.LoadUint64(&q.Queues[i].Weight)
		tw += ew
		if ew < mnw {
			mnw = ew
		}
	}
	for i := 0; i < 100; {
		for j := 0; j < len(q.Queues); j++ {
			rate := math.Round(float64(atomic.LoadUint64(&q.Queues[j].Weight)) / float64(mnw))
			mxp := int(rate)
			for k := 0; k < mxp; k++ {
				atomic.StoreUint32(&e.eprior[i], uint32(j))
				if i += 1; i == 100 {
					goto exit
				}
			}
		}
	}
exit:
	return
}

func (e *pq) shiftPQ() {
	for i := 0; i < len(e.pool); i++ {
		select {
		case itm, ok := <-e.pool[i]:
			if ok {
				e.mw().SubqPull(e.qn(uint32(i)))
				e.egress <- itm
				e.mw().SubqPut(qos.Egress)
				return
			}
		default:
			continue
		}
	}
}

func (e *pq) shiftRR() {
	qi := atomic.AddUint64(&e.rri, 1) % e.ql
	select {
	case itm, ok := <-e.pool[qi]:
		if ok {
			e.mw().SubqPull(e.qn(uint32(qi)))
			e.mw().SubqPut(qos.Egress)
			e.egress <- itm
		}
	default:
		return
	}
}

func (e *pq) shiftWRR() {
	pi := atomic.AddUint64(&e.wrri, 1) % 100
	qi := e.eprior[pi]
	select {
	case itm, ok := <-e.pool[qi]:
		if ok {
			e.mw().SubqPull(e.qn(qi))
			e.mw().SubqPut(qos.Egress)
			e.egress <- itm
		}
	default:
		return
	}
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

func (e *pq) qos() *qos.Config {
	return e.conf.QoS
}

func (e *pq) mw() MetricsWriter {
	return e.conf.MetricsWriter
}

func (e *pq) qn(i uint32) string {
	return e.qos().Queues[i].Name
}
