package qos

// PriorityEvaluator calculates priority percent of items comes to PQ.
type PriorityEvaluator interface {
	// Eval returns priority percent of x.
	Eval(x any) uint
}
