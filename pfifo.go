package queue

import (
	"math"
	"sync/atomic"
)

// Parallel FIFO engine implementation.
type pfifo struct {
	pool    []chan item
	c, o, m uint64
}

func (e *pfifo) init(config *Config) error {
	var inst uint64
	if inst = uint64(config.Streams); inst == 0 {
		inst = 1
	}
	cap_ := config.Capacity / inst
	for i := uint64(0); i < inst; i++ {
		e.pool = append(e.pool, make(chan item, cap_))
	}
	e.c, e.o, e.m = math.MaxUint64, math.MaxUint64, inst
	return nil
}

func (e *pfifo) enqueue(itm *item, block bool) bool {
	idx := atomic.AddUint64(&e.c, 1) % e.m
	if !block {
		select {
		case e.pool[idx] <- *itm:
			return true
		default:
			return false
		}
	}
	e.pool[idx] <- *itm
	return true
}

func (e *pfifo) dequeue() (item, bool) {
	idx := atomic.AddUint64(&e.o, 1) % e.m
	itm, ok := <-e.pool[idx]
	return itm, ok
}

func (e *pfifo) dequeueSQ(_ uint32) (item, bool) {
	return e.dequeue()
}

func (e *pfifo) size() (r int) {
	for i := uint64(0); i < e.m; i++ {
		r += len(e.pool[i])
	}
	return
}

func (e *pfifo) cap() (r int) {
	for i := uint64(0); i < e.m; i++ {
		r += cap(e.pool[i])
	}
	return
}

func (e *pfifo) close(_ bool) error {
	for i := uint64(0); i < e.m; i++ {
		close(e.pool[i])
	}
	return nil
}
