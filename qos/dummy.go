package qos

type DummyPriorityEvaluator struct{}

func (DummyPriorityEvaluator) Eval(_ any) uint { return 0 }
