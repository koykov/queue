package queue

import (
	"errors"
	"io"
)

type Pipeline interface {
	io.Closer
	Enqueuer
	AddQueue(q *Queue) Pipeline
	AddAsyncQueue(q *Queue) Pipeline
}

type pipeline struct {
	buf []pipeq
}

type pipeq struct {
	q     *Queue
	async bool
}

func NewPipeline() Pipeline {
	return &pipeline{}
}

func (p *pipeline) Enqueue(x any) error {
	// todo implement me
	return nil
}

func (p *pipeline) AddQueue(q *Queue) Pipeline {
	return p.add(q, false)
}

func (p *pipeline) AddAsyncQueue(q *Queue) Pipeline {
	return p.add(q, true)
}

func (p *pipeline) Close() error {
	var errs []error
	for i := 0; i < len(p.buf); i++ {
		if err := p.buf[i].q.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (p *pipeline) add(q *Queue, async bool) Pipeline {
	if q == nil {
		return p
	}
	q.donefn = p.donefn
	n := len(p.buf)
	if n > 0 {
		p.buf[n-1].q.nextq = q
	}
	p.buf = append(p.buf, pipeq{q: q, async: async})
	return p
}

func (p *pipeline) donefn(x any) {
	// todo design and implement
}
