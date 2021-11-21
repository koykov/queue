package blqueue

import (
	"strings"
	"sync"
)

type Schedule struct {
	ranges []shedRange
	once   sync.Once
}

type shedRange struct {
	l, r uint32
	n, x uint32
}

func NewSchedule() *Schedule {
	s := &Schedule{}
	s.once.Do(s.init)
	return s
}

func (s *Schedule) init() {
	//
}

func (s *Schedule) AddRange(raw string, min, max uint32) error {
	var pos int
	if pos = strings.Index(raw, "-"); pos == -1 {
		return ErrSchedBadRange
	}
	l, r := raw[:pos], raw[pos+1:]
	_, _ = l, r
	return nil
}
