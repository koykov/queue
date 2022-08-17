package blqueue

// WorkersChain describes workers list to process item consecutively.
type WorkersChain []Worker

// BindWorkers holds workers to one chain.
func BindWorkers(workers ...Worker) *WorkersChain {
	wc := WorkersChain(workers)
	return &wc
}

// Do process the item.
// Each worker in chain will be called for processing. Chain will stop processing on first failed worker.
func (w WorkersChain) Do(x interface{}) (err error) {
	for i := 0; i < len(w); i++ {
		if err = w[i].Do(x); err != nil {
			return
		}
	}
	return
}

var _ = BindWorkers
