package queue

type Job struct {
	Payload any
	Weight  uint64
}
