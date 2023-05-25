package qos

type Queue struct {
	Name     string
	Capacity uint64
	Weight   uint64
}
