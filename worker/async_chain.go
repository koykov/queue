package worker

import (
	"sync"
	"sync/atomic"

	"github.com/koykov/queue"
)

// AsyncChain describes workers list to asynchronously process item.
type AsyncChain []queue.Worker

// AsyncBind holds workers to one async chain.
func AsyncBind(workers ...queue.Worker) *AsyncChain {
	wc := AsyncChain{}
	return wc.Bind(workers...)
}

// Bind appends workers to the chain.
func (w *AsyncChain) Bind(workers ...queue.Worker) *AsyncChain {
	*w = append(*w, workers...)
	return w
}

// Do asynchronously process the item.
// Each worker in chain will be called for processing. AsyncChain will stop processing on first failed worker.
func (w AsyncChain) Do(x interface{}) (err error) {
	var (
		wg sync.WaitGroup
		ef uint32
	)
	for i := 0; i < len(w); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err1 := w[i].Do(x); err1 != nil {
				if atomic.AddUint32(&ef, 1) == 1 {
					err = err1
				}
				return
			}
		}(i)
	}
	wg.Wait()
	return
}

var _ = AsyncBind
