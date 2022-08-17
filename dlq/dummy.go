package dlq

// Dummy is a stub DLQ implementation. It does nothing and need for queues with leak tolerance.
// It just leaks data to the trash.
type Dummy struct{}

func (Dummy) Enqueue(_ interface{}) error { return nil }
func (Dummy) Rate() float32               { return 0 }
func (Dummy) Close() error                { return nil }
