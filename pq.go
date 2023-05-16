package queue

type pq struct {
	pool  []chan item
	prior [100]uint32
}

func (e *pq) init(config *Config) error {
	if config.QoS == nil {
		return ErrNoQoS
	}
	qos := config.QoS

	// Priorities buffer calculation.
	var tw uint64
	for i := 0; i < len(qos.Queues); i++ {
		tw += qos.Queues[i].Weight
	}
	var qi uint32
	for i := 0; i < len(qos.Queues); i++ {
		mxp := uint32(float64(qos.Queues[i].Weight) / float64(tw))
		for i := qi; i < mxp; i++ {
			e.prior[i] = i
		}
		qi = mxp
	}

	// Create channels.
	// ...

	// Start scheduler.
	// ...
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
