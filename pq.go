package queue

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/koykov/queue/qos"
)

// PQ (priority queuing) engine implementation.
type pq struct {
	subq    []chan item // sub-queues list
	egress  egress      // egress sub-queues
	inprior [100]uint32 // ingress priority table
	eprior  [100]uint32 // egress priority table (only for weighted algorithms)
	conf    *Config     // main config instance
	cancel  context.CancelFunc

	ew  int32         // active egress workers
	ewl int32         // locked egress workers
	ewc chan struct{} // egress workers control
	ia  int64         // idle attempts
	cp  uint64        // summing capacity
	ql  uint64        // sub-queues length
	rri uint64        // RR/WRR counter
}

func (e *pq) init(config *Config) error {
	if config.QoS == nil {
		return qos.ErrNoConfig
	}
	e.conf = config
	q := e.qos()
	e.cp = q.SummingCapacity()
	e.ql = uint64(len(q.Queues))
	e.rri = math.MaxUint64

	// Priorities tables calculation.
	e.rebalancePT()

	// Create channels.
	for i := 0; i < len(q.Queues); i++ {
		e.subq = append(e.subq, make(chan item, q.Queues[i].Capacity))
	}
	if err := e.egress.init(&config.QoS.Egress); err != nil {
		return err
	}
	e.ewc = make(chan struct{}, q.Egress.Workers)

	// Start egress worker(-s).
	var ctx context.Context
	ctx, e.cancel = context.WithCancel(context.Background())
	for i := uint32(0); i < q.Egress.Workers; i++ {
		atomic.AddInt32(&e.ew, 1)
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					atomic.AddInt32(&e.ew, -1)
					return
				default:
					var ok bool
					switch q.Algo {
					case qos.PQ:
						ok = e.shiftPQ()
					case qos.RR:
						ok = e.shiftRR()
					case qos.WRR:
						ok = e.shiftWRR()
					}
					if !ok {
						if atomic.AddInt64(&e.ia, 1) > int64(q.Egress.IdleThreshold) {
							// Too many idle recv attempts from sub-queues detected.
							// So lock EW till IdleTimeout reached or new item comes to the engine.
							select {
							case <-time.After(q.Egress.IdleTimeout):
								//
							case <-e.ewc:
								//
							}
						}
					}
				}
			}
		}(ctx)
	}
	return nil
}

func (e *pq) enqueue(itm *item, block bool) bool {
	// Evaluate priority.
	pp := e.qos().Evaluator.Eval(itm.payload)
	if pp == 0 {
		pp = 1
	}
	if pp > 100 {
		pp = 100
	}
	// Mark item with sub-queue index.
	itm.subqi = atomic.LoadUint32(&e.inprior[pp-1])
	q := e.subq[itm.subqi]
	qn := e.qn(itm.subqi)
	e.mw().SubqPut(qn)
	if !block {
		// Try to put item to the sub-queue in non-blocking mode.
		select {
		case q <- *itm:
			e.tryUnlockEW()
			return true
		default:
			e.mw().SubqLeak(qn)
			return false
		}
	} else {
		// ... or in blocking mode.
		q <- *itm
		e.tryUnlockEW()
	}

	return true
}

// Try to send unlock signal to all active EW.
func (e *pq) tryUnlockEW() {
	atomic.StoreInt64(&e.ia, 0)
	if atomic.LoadInt64(&e.ia) > int64(e.qos().Egress.IdleThreshold) {
		for i := 0; i < int(atomic.LoadInt32(&e.ew)); i++ {
			select {
			case e.ewc <- struct{}{}:
				//
			default:
				//
			}
		}
	}
}

func (e *pq) dequeue() (item, bool) {
	itm, eqi, ok := e.egress.dequeue()
	if ok {
		e.mw().SubqPull(e.egress.qn(eqi))
	}
	return itm, ok
}

func (e *pq) dequeueSQ(subqi uint32) (item, bool) {
	itm, ok := <-e.subq[subqi]
	if ok {
		e.mw().SubqPull(e.qn(subqi))
	}
	return itm, ok
}

func (e *pq) size() (sz int) {
	return e.size1(true)
}

// Internal size evaluator.
func (e *pq) size1(includingEgress bool) (sz int) {
	for i := 0; i < len(e.subq); i++ {
		sz += len(e.subq[i])
	}
	if includingEgress {
		sz += e.egress.size()
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
	e.tryUnlockEW()
	e.cancel()
	// Close sub-queues channels.
	for i := 0; i < len(e.subq); i++ {
		close(e.subq[i])
	}
	// Spinlock waiting till all egress workers finished; close control channel.
	for atomic.LoadInt32(&e.ew) > 0 {
	}
	close(e.ewc)
	// Close egress channels.
	return e.egress.close()
}

// Priority tables (ingress and egress) rebalance.
func (e *pq) rebalancePT() {
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
		tw += atomic.LoadUint64(&q.Queues[i].IngressWeight)
	}
	var qi uint32
	for i := 0; i < len(q.Queues); i++ {
		rate := math.Ceil(float64(atomic.LoadUint64(&q.Queues[i].IngressWeight)) / float64(tw) * 100)
		mxp := uint32(rate)
		for j := qi; j < mxu32(qi+mxp, 100); j++ {
			atomic.StoreUint32(&e.inprior[lim(j, 99)], uint32(i))
		}
		qi += mxp
	}

	// Build and shuffle egress priority table.
	var mnw uint64 = math.MaxUint64
	for i := 0; i < len(q.Queues); i++ {
		ew := atomic.LoadUint64(&q.Queues[i].EgressWeight)
		tw += ew
		if ew < mnw {
			mnw = ew
		}
	}
	for i := 0; i < 100; {
		for j := 0; j < len(q.Queues); j++ {
			rate := math.Round(float64(atomic.LoadUint64(&q.Queues[j].EgressWeight)) / float64(mnw))
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

// PQ algorithm implementation: try to recv one single item from first available sub-queue (considering order) and send
// it to egress.
func (e *pq) shiftPQ() bool {
	for i := 0; i < len(e.subq); i++ {
		select {
		case itm, ok := <-e.subq[i]:
			if ok {
				e.mw().SubqPull(e.qn(uint32(i)))
				eqi := e.egress.enqueue(itm)
				e.mw().SubqPut(e.egress.qn(eqi))
				return true
			}
		default:
			continue
		}
	}
	return false
}

// RR algorithm implementation: try to recv one single item from sequential sub-queue and send it to egress.
func (e *pq) shiftRR() bool {
	qi := atomic.AddUint64(&e.rri, 1) % e.ql // sub-queue index trick.
	select {
	case itm, ok := <-e.subq[qi]:
		if ok {
			e.mw().SubqPull(e.qn(uint32(qi)))
			eqi := e.egress.enqueue(itm)
			e.mw().SubqPut(e.egress.qn(eqi))
			return true
		}
	default:
		return false
	}
	return false
}

// WRR/DWRR algorithm implementation: try to recv one single item from sequential sub-queue (considering weight) and
// send it to egress.
func (e *pq) shiftWRR() bool {
	pi := atomic.AddUint64(&e.rri, 1) % 100 // PT weight trick.
	qi := e.eprior[pi]
	select {
	case itm, ok := <-e.subq[qi]:
		if ok {
			e.mw().SubqPull(e.qn(qi))
			eqi := e.egress.enqueue(itm)
			e.mw().SubqPut(e.egress.qn(eqi))
			return true
		}
	default:
		return false
	}
	return false
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

// egress stream implementation
type egress struct {
	pool    []chan item // pool of egress sub-queues
	name    []string    // names of egress sub-queues
	c, o, m uint64
}

func (e *egress) init(conf *qos.EgressConfig) error {
	for i := uint32(0); i < conf.Streams; i++ {
		e.pool = append(e.pool, make(chan item, conf.Capacity))
		name := qos.Egress
		if conf.Streams > 1 {
			name = fmt.Sprintf("%s%d", qos.Egress, i)
		}
		e.name = append(e.name, name)
	}
	e.c, e.o, e.m = math.MaxUint64, math.MaxUint64, uint64(conf.Streams)
	return nil
}

func (e *egress) enqueue(itm item) uint64 {
	idx := atomic.AddUint64(&e.c, 1) % e.m
	e.pool[idx] <- itm
	return idx
}

func (e *egress) dequeue() (item, uint64, bool) {
	for i := 0; i < len(e.pool); i++ {
		idx := atomic.AddUint64(&e.o, 1) % e.m
		itm, ok := <-e.pool[idx]
		if ok {
			return itm, idx, ok
		}
	}
	return item{}, 0, false
}

func (e *egress) size() (sz int) {
	for i := 0; i < len(e.pool); i++ {
		sz += len(e.pool[i])
	}
	return
}

func (e *egress) close() error {
	for i := 0; i < len(e.pool); i++ {
		close(e.pool[i])
	}
	return nil
}

func (e *egress) qn(idx uint64) string {
	return e.name[idx]
}
