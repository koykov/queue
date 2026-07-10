package queue

import "io"

type Pipeline interface {
	io.Closer
	Enqueuer
	AddQueue(q *Queue) Pipeline
	AddAsyncQueue(q *Queue) Pipeline
	AddPipeline(p Pipeline) Pipeline
	AddAsyncPipeline(p Pipeline) Pipeline
}

type pipeline struct {
	// ...
}

func (p *pipeline) Enqueue(x any) error {
	// todo implement me
	return nil
}
