package worker

import "github.com/koykov/queue"

// Chain describes workers list to process item consecutively.
type Chain []queue.Worker

// Bind holds workers to one chain.
func Bind(workers ...queue.Worker) *Chain {
	wc := Chain{}
	return wc.Bind(workers...)
}

// Bind appends workers to the chain.
func (w *Chain) Bind(workers ...queue.Worker) *Chain {
	*w = append(*w, workers...)
	return w
}

// Do process the item.
// Each worker in chain will be called for processing. Chain will stop processing on first failed worker.
func (w *Chain) Do(x any) (err error) {
	for i := 0; i < len(*w); i++ {
		if err = (*w)[i].Do(x); err != nil {
			return
		}
	}
	return
}

var _ = Bind
