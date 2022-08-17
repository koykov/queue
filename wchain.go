package blqueue

// WorkersChain describes workers list to process item consecutively.
type WorkersChain []Worker

func BindWorkers(workers ...Worker) *WorkersChain {
	wc := WorkersChain(workers)
	return &wc
}

func (w WorkersChain) Do(x interface{}) (err error) {
	for i := 0; i < len(w); i++ {
		if err = w[i].Do(x); err != nil {
			return
		}
	}
	return
}

var _ = BindWorkers
