package blqueue

import (
	"testing"
	"time"
	"unsafe"
)

func TestConfig(t *testing.T) {
	t.Run("copy", func(t *testing.T) {
		cmpDeepPtr := func(a, b *Config) (bool, bool) {
			ap, bp := uintptr(unsafe.Pointer(&a)), uintptr(unsafe.Pointer(&b))
			asp, bsp := uintptr(unsafe.Pointer(a.Schedule)), uintptr(unsafe.Pointer(b.Schedule))
			return ap == bp, asp == bsp
		}
		sched := NewSchedule()
		_ = sched.AddRange("12:00-13:00", ScheduleParams{4, 16, .25, .01})
		_ = sched.AddRange("06:12:43-08:43:12", ScheduleParams{1, 4, .7, .25})
		_ = sched.AddRange("16:00:30-18:05:32", ScheduleParams{2, 8, .15, .05})
		conf := &Config{
			Key:          "foobar",
			Size:         1e5,
			Heartbeat:    time.Minute,
			WorkersMin:   1,
			WorkersMax:   10,
			WakeupFactor: .5,
			SleepFactor:  .1,
			SleepTimeout: time.Minute,
			Schedule:     sched,
		}
		cpy := conf.Copy()
		if ce, se := cmpDeepPtr(conf, cpy); ce || se {
			if ce {
				t.Errorf("config and copy has the same address")
			}
			if se {
				t.Errorf("schedule and copy has the same address")
			}
		}
	})
}
