package queue

import (
	"sort"
	"testing"
)

func TestPQ(t *testing.T) {
	t.Run("priority table", func(t *testing.T) {
		expectPB := [100]uint32{
			0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
		}
		conf := Config{
			QoS: NewQoS(RR, DummyPriorityEvaluator{}).AddNamedQueue("low", 750, 1200).
				AddNamedQueue("high", 200, 120).
				AddNamedQueue("medium", 50, 400),
		}
		sort.Sort(conf.QoS)
		q := pq{}
		err := q.init(&conf)
		if err != nil {
			t.Error(err)
		}
		if i, ok := q.assertPB(expectPB); !ok {
			t.Errorf("PB mismatch at position %d", i)
		}
	})
}
