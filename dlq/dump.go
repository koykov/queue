package dlq

import "time"

const (
	Byte     MemorySize = 1
	Kilobyte            = Byte * 1024
	Megabyte            = Kilobyte * 1024
	Gigabyte            = Megabyte * 1024
	Terabyte            = Gigabyte * 1024
	_                   = Terabyte
)

type MemorySize uint64

type Dump struct {
	Size      MemorySize
	TimeLimit time.Duration
}

func (q *Dump) Enqueue(x interface{}) error {
	_ = x
	return nil
}

func (q Dump) Rate() float32 {
	return 0
}

func (q Dump) Close() error {
	return nil
}
