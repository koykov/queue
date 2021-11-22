package blqueue

import "testing"

func TestSchedule(t *testing.T) {
	t.Run("min > max", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("foobar", 10, 8); err != ErrSchedMinGtMax {
			t.Errorf("bad error: need %s, got %s", ErrSchedMinGtMax, err)
		}
	})
	t.Run("bad range", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("foobar", 0, 0); err != ErrSchedBadRange {
			t.Errorf("bad error: need %s, got %s", ErrSchedBadRange, err)
		}
	})
	t.Run("bad range -20:00:00", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("-20:00:00", 0, 0); err != ErrSchedBadRange {
			t.Errorf("bad error: need %s, got %s", ErrSchedBadRange, err)
		}
	})
	t.Run("bad range 06:00:00-", func(t *testing.T) {
		s := NewSchedule()
		if err := s.AddRange("06:00:00-", 0, 0); err != ErrSchedBadRange {
			t.Errorf("bad error: need %s, got %s", ErrSchedBadRange, err)
		}
	})
	t.Run("08:30:00-14:00:05", func(t *testing.T) {
		s := NewSchedule()
		_ = s.AddRange("08:30:00-14:00:05", 0, 0)
	})
}
