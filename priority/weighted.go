package priority

import (
	"sync"
)

// Weighted priority evaluator.
type Weighted struct {
	Default uint
	Max     uint

	once sync.Once
	max  float64
	def  uint
}

func (e *Weighted) Eval(weight any) uint {
	e.once.Do(e.init)
	var w float64
	switch weight.(type) {
	case int:
		w = float64(weight.(int))
	case int8:
		w = float64(weight.(int8))
	case int16:
		w = float64(weight.(int16))
	case int32:
		w = float64(weight.(int32))
	case int64:
		w = float64(weight.(int64))

	case uint:
		w = float64(weight.(uint))
	case uint8:
		w = float64(weight.(uint8))
	case uint16:
		w = float64(weight.(uint16))
	case uint32:
		w = float64(weight.(uint32))
	case uint64:
		w = float64(weight.(uint64))

	case float32:
		w = float64(weight.(float32))
	case float64:
		w = weight.(float64)
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
