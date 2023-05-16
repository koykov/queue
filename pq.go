package queue

type pq struct {
	pool  []chan item
	prior [100]uint
}

func (e *pq) init(config *Config) error {
	if config.QoS == nil {
		return ErrNoQoS
	}
	return nil
}

func (e *pq) put(itm *item, block bool) bool {
	return true
}

func (e *pq) getc() chan item {
	return nil
}

func (e *pq) size() int {
	return 0
}

func (e *pq) cap() int {
	return 0
}

func (e *pq) close(force bool) error {
	return nil
}
