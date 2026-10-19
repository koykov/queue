package queue

// ErrorRater describes a processing error rate handler.
type ErrorRater interface {
	// Report registers the fact of item processing with failed flag.
	Report(failed bool)
	// Rate returns the current error rate in range [0,1].
	Rate() float32
}

type dummyRater struct{}

func (dummyRater) Report(_ bool) {}
func (dummyRater) Rate() float32 { return 0 }
