package queue

type Job struct {
	Payload any
	Weight  uint64
}

func tryJob(x any) (itm item) {
	switch x.(type) {
	case Job:
		job := x.(Job)
		itm.payload, itm.weight = job.Payload, job.Weight
	case *Job:
		job := x.(*Job)
		itm.payload, itm.weight = job.Payload, job.Weight
	default:
		itm.payload = x
	}
	return
}
