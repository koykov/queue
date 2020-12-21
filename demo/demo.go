package main

import (
	"bytes"
	"encoding/json"

	"github.com/koykov/queue"
)

type demoQueue struct {
	queue queue.Queuer

	producersMin,
	producersMax uint32
	producers []producer
	ctl       []chan uint8
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
