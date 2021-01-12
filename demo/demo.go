package main

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/koykov/queue"
)

type demoQueue struct {
	queue *queue.Queue

	producersMin,
	producersMax,
	producersUp uint32
	producers []producer
	ctl       []chan signal
}

func (d *demoQueue) Run() {
	for i := 0; i < int(d.producersMin); i++ {
		d.ctl[i] = make(chan signal, 1)
		d.producers[i].idx = uint32(i)
		go d.producers[i].produce(d.queue, d.ctl[i])
		d.ctl[i] <- signalInit
	}
	d.producersUp = d.producersMin
}

func (d *demoQueue) ProducerUp() error {
	i := d.producersUp
	if i == d.producersMax {
		return errors.New("maximum producers count reached")
	}
	d.producers[i].idx = i
	d.ctl[i] = make(chan signal, 1)
	go d.producers[i].produce(d.queue, d.ctl[i])
	d.ctl[i] <- signalInit
	d.producersUp++
	return nil
}

func (d *demoQueue) String() string {
	var out = &struct {
		Queue           string `json:"queue"`
		ProducersMin    int    `json:"producers_min"`
		ProducersMax    int    `json:"producers_max"`
		ProducersIdle   int    `json:"producers_idle"`
		ProducersActive int    `json:"producers_active"`
	}{}

	out.Queue = "!queue"
	out.ProducersMin = int(d.producersMin)
	out.ProducersMax = int(d.producersMax)
	for _, p := range d.producers {
		switch p.status {
		case statusIdle:
			out.ProducersIdle++
		case statusActive:
			out.ProducersActive++
		}
	}

	b, _ := json.Marshal(out)
	b = bytes.Replace(b, []byte(`"!queue"`), []byte(d.queue.String()), 1)

	return string(b)
}
