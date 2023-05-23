package queue

// FIFO engine implementation.
type fifo struct {
	c chan item
}

func (e *fifo) init(config *Config) error {
	e.c = make(chan item, config.Capacity)
	return nil
}

func (e *fifo) enqueue(itm *item, block bool) bool {
	if !block {
		select {
		case e.c <- *itm:
			return true
		default:
			return false
		}
	}
	e.c <- *itm
	return true
}

func (e *fifo) dequeue(block bool) (item, bool) {
	if block {
		itm := <-e.c
		return itm, true
	}
	itm, ok := <-e.c
	return itm, ok
}

func (e *fifo) dequeueSQ(_ uint32, block bool) (item, bool) {
	return e.dequeue(block)
}

func (e *fifo) size() int {
	return len(e.c)
}

func (e *fifo) cap() int {
	return cap(e.c)
}

func (e *fifo) close(_ bool) error {
	close(e.c)
	return nil
}
