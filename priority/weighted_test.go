package priority

import (
	"testing"
)

var stagesWeighted = []struct{ weight, priority uint }{
	{0, 50},
	{1, 0},
	{5, 0},
	{10, 0},
	{50, 3},
	{100, 6},
	{133, 8},
	{166, 11},
	{200, 13},
	{277, 18},
	{350, 23},
	{413, 27},
	{499, 33},
	{502, 33},
	{689, 45},
	{777, 51},
	{801, 53},
	{999, 66},
	{1000, 66},
	{1001, 66},
	{1077, 71},
	{1305, 87},
	{1403, 93},
	{1495, 99},
	{1500, 100},
	{1501, 100},
	{1666, 100},
	{1999, 100},
	{2000, 100},
}

func TestWeighted(t *testing.T) {
	e := Weighted{Max: 1500, Default: 50}
	for _, stage := range stagesWeighted {
		p := e.Eval(stage.weight)
		if p != stage.priority {
			t.Errorf("priority mismatch: need %d, got %d", stage.priority, p)
		}
	}
}

func BenchmarkWeighted(b *testing.B) {
	b.Run("single", func(b *testing.B) {
		l := len(stagesWeighted)
		e := Weighted{Max: 1500, Default: 50}
		for i := 0; i < b.N; i++ {
			stage := stagesWeighted[i%l]
			p := e.Eval(stage.weight)
			if p != stage.priority {
				b.Errorf("priority mismatch: need %d, got %d", stage.priority, p)
			}
		}
	})
	b.Run("parallel", func(b *testing.B) {
		for _, stage := range stagesWeighted {
			e := Weighted{Max: 1500, Default: 50}
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					p := e.Eval(stage.weight)
					if p != stage.priority {
						b.Errorf("priority mismatch: need %d, got %d", stage.priority, p)
					}
				}
			})
		}
	})
}
