package main

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/koykov/blqueue"
)

type status uint32
type signal uint32

const (
	statusIdle   status = 0
	statusActive status = 1

	signalInit signal = 0
	signalStop signal = 1
)

type producer struct {
	idx    uint32
	ctl    chan signal
	status status
}

func makeProducer(idx uint32) *producer {
	p := &producer{
		idx:    idx,
		ctl:    make(chan signal, 1),
		status: statusIdle,
	}
	return p
}

func (p *producer) start() {
	p.ctl <- signalInit
}

func (p *producer) stop() {
	p.ctl <- signalStop
}

func (p *producer) produce(q *blqueue.Queue) {
	for {
		select {
		case cmd := <-p.ctl:
			switch cmd {
			case signalInit:
				p.setStatus(statusActive)
			case signalStop:
				p.setStatus(statusIdle)
				return
			}
		default:
			if p.getStatus() == statusIdle {
				return
			}
			x := struct {
				Header  uint32
				Payload int64
			}{4, math.MaxInt64}
			time.Sleep(time.Nanosecond * 50)
			q.Enqueue(x)
		}
	}
}

func (p *producer) setStatus(status status) {
	atomic.StoreUint32((*uint32)(&p.status), uint32(status))
}

func (p *producer) getStatus() status {
	return status(atomic.LoadUint32((*uint32)(&p.status)))
}
