package blqueue

import "testing"

func TestSchedule(t *testing.T) {
	t.Run("min > max", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("foobar", 10, 8, 0, 0); err != ErrSchedMinGtMax {
			t.Errorf("bad error: need %s, got %s", ErrSchedMinGtMax, err)
		}
	})
	t.Run("bad range", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("foobar", 0, 1, 0, 0); err != ErrSchedBadRange {
			t.Errorf("bad error: need %s, got %s", ErrSchedBadRange, err)
		}
	})
	t.Run("bad range -20:00:00", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("-20:00:00", 0, 1, 0, 0); err != ErrSchedBadRange {
			t.Errorf("bad error: need %s, got %s", ErrSchedBadRange, err)
		}
	})
	t.Run("bad range 06:00:00-", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("06:00:00-", 0, 1, 0, 0); err != ErrSchedBadRange {
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
			if err := s.AddRange(r, 0, 1, .5, .1); err != nil {
				t.Error(err)
			}
			if s.String() != exp {
				t.Error("schedule failed")
			}
		}
	})
}
