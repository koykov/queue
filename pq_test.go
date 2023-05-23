package queue

import (
	"testing"
)

func TestPQ(t *testing.T) {
	t.Run("priority table", func(t *testing.T) {
		expectPT := [100]uint32{
			0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
		}
		conf := Config{
			QoS: NewQoS(RR, DummyPriorityEvaluator{}).
				AddNamedQueue("high", 200, 120).
				AddNamedQueue("medium", 50, 400).
				AddNamedQueue("low", 750, 1200),
		}
		q := pq{}
		err := q.init(&conf)
		if err != nil {
			t.Error(err)
		}
		if i, ok := q.assertPT(expectPT); !ok {
			t.Errorf("PT mismatch at position %d", i)
		}
	})
}
