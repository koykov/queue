package queue

// FIFO engine implementation.
type fifo struct {
	c chan item
}

func (e *fifo) init(config *Config) error {
	e.c = make(chan item, config.Capacity)
	return nil
}

func (e *fifo) put(itm *item, block bool) bool {
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

func (e *fifo) getc() chan item {
	return e.c
}

func (e *fifo) size() int {
	return len(e.c)
}

func (e *fifo) cap() int {
	return cap(e.c)
}

func (e *fifo) close() error {
	close(e.c)
	return nil
}
