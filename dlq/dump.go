package dlq

import "time"

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

type EncoderDecoder interface {
	Encoder
	Decoder
}

type Dump struct {
	Size           MemorySize
	TimeLimit      time.Duration
	EncoderDecoder EncoderDecoder
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
