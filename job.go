package queue

// Job is a wrapper about queue item. May provide meta info.
type Job struct {
	// Item payload.
	Payload any
	// Item weight. Designed to use together with Weighted priority evaluator (see priority/weighted.go).
	Weight uint64
}
