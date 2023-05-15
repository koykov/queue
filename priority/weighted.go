package priority

import (
	"sync"
)

// Weighted priority evaluator.
type Weighted struct {
	Default uint
	Max     uint

	once     sync.Once
	max, def uint
}

func (e *Weighted) Eval(weight any) uint {
	e.once.Do(e.init)
	var w uint
	switch weight.(type) {
	case int:
		w = uint(weight.(int))
	case int8:
		w = uint(weight.(int8))
	case int16:
		w = uint(weight.(int16))
	case int32:
		w = uint(weight.(int32))
	case int64:
		return uint(weight.(int64))

	case uint:
		w = weight.(uint)
	case uint8:
		w = uint(weight.(uint8))
	case uint16:
		w = uint(weight.(uint16))
	case uint32:
		w = uint(weight.(uint32))
	case uint64:
		w = uint(weight.(uint64))

	case float32:
		w = uint(weight.(float32))
	case float64:
		w = uint(weight.(float64))
	}

	if w == 0 {
		w = e.Default
	}
	if w >= e.max {
		return 100
	}
	return w / e.max * 100
}

func (e *Weighted) init() {
	e.max = e.Max
	e.def = e.Default
}
