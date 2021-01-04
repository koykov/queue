package main

import (
	"math"

	"github.com/koykov/queue"
)

type status uint
type signal uint

const (
	statusIdle   status = 0
	statusActive        = 1
	statusStop          = 2

	signalInit  signal = 0
	signalSleep        = 1
	signalStop         = 2
)

type producer struct {
	idx    uint32
	status status
}

func (p *producer) produce(q *queue.Queue1, ctl chan signal) {
	for {
		select {
		case cmd := <-ctl:
			switch cmd {
			case signalInit:
				p.status = statusActive
			case signalSleep:
				p.status = statusIdle
			case signalStop:
				p.status = statusStop
				return
			}
		default:
			if p.status == statusIdle {
				continue
			}
			if p.status == statusStop {
				break
			}
			x := struct {
				Header  uint32
				Payload int64
			}{4, math.MaxInt64}
			q.Enqueue(x)
		}
	}
}
