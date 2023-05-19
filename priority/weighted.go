package priority

import (
	"sync"

	"github.com/koykov/queue"
)

// Weighted priority evaluator.
type Weighted struct {
	Default uint
	Max     uint

	once sync.Once
	max  float64
	def  uint
}

func (e *Weighted) Eval(x any) uint {
	e.once.Do(e.init)
	var w float64
	switch x.(type) {
	case queue.Job:
		job := x.(queue.Job)
		w = float64(job.Weight)
	case *queue.Job:
		job := x.(*queue.Job)
		w = float64(job.Weight)

	case int:
		w = float64(x.(int))
	case int8:
		w = float64(x.(int8))
	case int16:
		w = float64(x.(int16))
	case int32:
		w = float64(x.(int32))
	case int64:
		w = float64(x.(int64))

	case uint:
		w = float64(x.(uint))
	case uint8:
		w = float64(x.(uint8))
	case uint16:
		w = float64(x.(uint16))
	case uint32:
		w = float64(x.(uint32))
	case uint64:
		w = float64(x.(uint64))

	case float32:
		w = float64(x.(float32))
	case float64:
		w = x.(float64)
	}

	if w == 0 {
		return e.def
	}
	if w >= e.max {
		return 100
	}
	return uint(w / e.max * 100)
}

func (e *Weighted) init() {
	if e.max = float64(e.Max); e.max == 0 {
		e.max = 100
	}
	e.def = e.Default
}
