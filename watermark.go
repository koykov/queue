package queue

import (
	"math"
	"sync/atomic"
)

type watermark int64

func (w *watermark) set(val uint64) {
	atomic.AddInt64((*int64)(w), int64(val-math.MaxInt32))
}

func (w *watermark) inc() {
	atomic.AddInt64((*int64)(w), 1)
}

func (w *watermark) dec() {
	atomic.AddInt64((*int64)(w), -1)
}

func (w *watermark) add(delta uint64) {
	atomic.AddInt64((*int64)(w), int64(delta-math.MaxInt32))
}

func (w *watermark) get() uint64 {
	return uint64(atomic.LoadInt64((*int64)(w)) + math.MaxInt32)
}
