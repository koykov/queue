package queue

type Rater interface {
	Report(status bool)
	Rate() float32
}

type dummyRater struct{}

func (dummyRater) Report(_ bool) {}
func (dummyRater) Rate() float32 { return 0 }
