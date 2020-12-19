package main

import "github.com/koykov/queue"

type demoQueue struct {
	queue queue.Queuer

	producersMin,
	producersMax uint32
	producers []producerProc
	ctl       []chan uint8
}

func (d *demoQueue) String() string {
	return ""
}
