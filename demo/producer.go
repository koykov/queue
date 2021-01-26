package main

import (
	"math"
	"time"

	"github.com/koykov/queue"
)

type status uint
type signal uint

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

func (p *producer) produce(q *queue.Queue) {
	for {
		select {
		case cmd := <-p.ctl:
			switch cmd {
			case signalInit:
				p.status = statusActive
			case signalStop:
				p.status = statusIdle
				return
			}
		default:
			if p.status == statusIdle {
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
