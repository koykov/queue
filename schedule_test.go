package blqueue

import "testing"

func TestSchedule(t *testing.T) {
	t.Run("min > max", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("foobar", RealtimeParams{10, 8, 0, 0}); err != ErrSchedMinGtMax {
			t.Errorf("bad error: need %s, got %s", ErrSchedMinGtMax, err)
		}
	})
	t.Run("bad range", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("foobar", RealtimeParams{0, 1, 0, 0}); err != ErrSchedBadRange {
			t.Errorf("bad error: need %s, got %s", ErrSchedBadRange, err)
		}
	})
	t.Run("bad range -20:00:00", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("-20:00:00", RealtimeParams{0, 1, 0, 0}); err != ErrSchedBadRange {
			t.Errorf("bad error: need %s, got %s", ErrSchedBadRange, err)
		}
	})
	t.Run("bad range 06:00:00-", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("06:00:00-", RealtimeParams{0, 1, 0, 0}); err != ErrSchedBadRange {
			t.Errorf("bad error: need %s, got %s", ErrSchedBadRange, err)
		}
	})
	t.Run("08:30:00-14:00:05", func(t *testing.T) {
		exp := `[
	08:30:00.000-14:00:05.000 min: 0 max: 1 wakeup: 0.5 sleep: 0.1
]`
		rules := []string{
			"08:30-14:00:05",
			"08:30:00.000-14:00:05",
			"08:30:00-14:00:05.000",
		}
		for _, r := range rules {
			s := NewSchedule()
			if err := s.AddRange(r, RealtimeParams{0, 1, .5, .1}); err != nil {
				t.Error(err)
			}
			if s.String() != exp {
				t.Error("schedule failed")
			}
		}
	})
	t.Run("sort", func(t *testing.T) {
		exp := `[
	06:12:43.000-08:43:12.000 min: 0 max: 1 wakeup: 0 sleep: 0
	12:00:00.000-13:00:00.000 min: 0 max: 1 wakeup: 0 sleep: 0,
	16:00:30.000-18:05:32.000 min: 0 max: 1 wakeup: 0 sleep: 0,
]`
		s := NewSchedule()
		_ = s.AddRange("12:00-13:00", RealtimeParams{0, 1, 0, 0})
		_ = s.AddRange("06:12:43-08:43:12", RealtimeParams{0, 1, 0, 0})
		_ = s.AddRange("16:00:30-18:05:32", RealtimeParams{0, 1, 0, 0})
		if s.String() != exp {
			t.Error("sort failed")
		}
	})
}
