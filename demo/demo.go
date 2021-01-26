package main

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/koykov/queue"
)

type demoQueue struct {
	key   string
	queue *queue.Queue

	producersMin,
	producersMax,
	producersUp uint32
	producers []*producer
}

func (d *demoQueue) Run() {
	d.producers = make([]*producer, d.producersMax)
	for i := 0; i < int(d.producersMax); i++ {
		d.producers[i] = makeProducer(uint32(i))
	}
	for i := 0; i < int(d.producersMin); i++ {
		go d.producers[i].produce(d.queue)
		d.producers[i].start()
	}
	d.producersUp = d.producersMin

	producerActive.WithLabelValues(d.key).Add(float64(d.producersUp))
	producerIdle.WithLabelValues(d.key).Add(float64(d.producersMax - d.producersUp))
}

func (d *demoQueue) ProducerUp(delta uint32) error {
	if delta == 0 {
		delta = 1
	}
	if d.producersUp+delta >= d.producersMax {
		return errors.New("maximum producers count reached")
	}
	c := d.producersUp
	for i := c; i < c+delta; i++ {
		go d.producers[i].produce(d.queue)
		d.producers[i].start()
		d.producersUp++
		ProducerStartMetric(d.key)
	}
	return nil
}

func (d *demoQueue) ProducerDown(delta uint32) error {
	if delta == 0 {
		delta = 1
	}
	if d.producersUp-delta < d.producersMin {
		return errors.New("minimum producers count reached")
	}
	c := d.producersUp
	for i := c; i >= c-delta; i-- {
		if d.producers[i].status == statusActive {
			d.producers[i].stop()
			d.producersUp--
			ProducerStopMetric(d.key)
		}
	}
	return nil
}

func (d *demoQueue) Stop() {
	c := d.producersUp
	for i := uint32(0); i < c; i++ {
		d.producers[i].stop()
		d.producersUp--
		ProducerStopMetric(d.key)
	}
	d.queue.Close()
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
