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
	p.buf = append(p.buf, pipeq{q: q})
	return p
}

func (p *pipeline) AddAsyncQueue(q *Queue) Pipeline {
	p.buf = append(p.buf, pipeq{q: q, async: true})
	return p
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
