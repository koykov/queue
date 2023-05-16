package queue

import (
	"sort"
	"testing"
)

func TestQoS(t *testing.T) {
	t.Run("sort", func(t *testing.T) {
		mustName := func(t *testing.T, q *QoSQueue, r string) {
			if q.Name != r {
				t.Errorf("name mismatch: need '%s', got '%s'", r, q.Name)
			}
		}
		qos := NewQoS(PQ, DummyPriorityEvaluator{}).AddNamedQueue("low", 750, 1200).
			AddNamedQueue("high", 200, 120).
			AddNamedQueue("medium", 50, 400)
		sort.Sort(qos)
		mustName(t, &qos.Queues[0], "high")
		mustName(t, &qos.Queues[1], "medium")
		mustName(t, &qos.Queues[2], "low")
	})
}
