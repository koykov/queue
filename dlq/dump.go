package dlq

import (
	"math"
	"sync"
	"time"
)

type MemorySize uint64

const (
	Byte     MemorySize = 1
	Kilobyte            = Byte * 1024
	Megabyte            = Kilobyte * 1024
	Gigabyte            = Megabyte * 1024
	Terabyte            = Gigabyte * 1024
	_                   = Terabyte
)

type Encoder interface {
	Encode(dst []byte, x interface{}) ([]byte, error)
}

type Decoder interface {
	Decode(p []byte) (interface{}, error)
}

type Dump struct {
	Size      MemorySize
	TimeLimit time.Duration
	Encoder   Encoder
	Decoder   Decoder

	mux sync.Mutex
	buf []byte
}

func (q *Dump) Enqueue(x interface{}) (err error) {
	q.mux.Lock()
	defer q.mux.Unlock()

	// off := len(q.buf)
	if q.Encoder != nil {
		if q.buf, err = q.Encoder.Encode(q.buf, x); err != nil {
			return
		}
	} else {
		// todo check different types of x
	}

	return
}

func (q *Dump) Rate() float32 {
	q.mux.Lock()
	defer q.mux.Unlock()
	if q.Size == 0 {
		return math.MaxFloat32
	}
	return float32(len(q.buf)) / float32(q.Size)
}

func (q *Dump) Close() error {
	q.mux.Lock()
	defer q.mux.Unlock()
	return nil
}
