package queue

import (
	"testing"
)

func TestPQ(t *testing.T) {
	t.Run("priority table", func(t *testing.T) {
		expectIPT := [100]uint32{
			0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
		}
		expectEPT := [100]uint32{
			0, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 0, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
			0, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 0, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
			0, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 0, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
			0, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 0, 1,
		}
		conf := Config{
			QoS: NewQoS(RR, DummyPriorityEvaluator{}).
				AddQueue(QoSQueue{Name: "high", Capacity: 200, Weight: 120}).
				AddQueue(QoSQueue{Name: "medium", Capacity: 50, Weight: 400}).
				AddQueue(QoSQueue{Name: "low", Capacity: 750, Weight: 1200}),
		}
		_ = conf.QoS.Validate()
		q := pq{}
		err := q.init(&conf)
		if err != nil {
			t.Error(err)
		}
		if i, ok := q.assertPT(expectIPT, expectEPT); !ok {
			t.Errorf("PT mismatch at position %d", i)
		}
		_ = q.close(false)
	})
}
