package blqueue

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	msItself = 1
	msSec    = 1000 * msItself
	msMin    = 60 * msSec
	msHour   = 60 * msMin
)

type Schedule struct {
	ranges []schedRange
	once   sync.Once
}

type schedRange struct {
	l, r uint32
	n, x uint32
}

var (
	reHMSMs = regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2})\.(\d{3})`)
	reHMS   = regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2})`)
	reHM    = regexp.MustCompile(`(\d{2}):(\d{2})`)
)

func NewSchedule() *Schedule {
	s := &Schedule{}
	s.once.Do(s.init)
	return s
}

func (s *Schedule) init() {
	// todo implement me
}

func (s *Schedule) AddRange(raw string, min, max uint32) (err error) {
	if max == 0 {
		return ErrSchedZeroMax
	}
	if min > max {
		return ErrSchedMinGtMax
	}

	var pos int
	if pos = strings.Index(raw, "-"); pos == -1 {
		return ErrSchedBadRange
	}
	l, r := raw[:pos], raw[pos+1:]
	if len(l) == 0 || len(r) == 0 {
		return ErrSchedBadRange
	}

	var (
		lt, rt uint32
	)
	if lt, err = s.parse(l, 0); err != nil {
		return err
	}
	if rt, err = s.parse(r, 1); err != nil {
		return err
	}
	if rt < lt {
		return ErrSchedBadRange
	}
	s.ranges = append(s.ranges, schedRange{
		l: lt,
		r: rt,
		n: min,
		x: max,
	})
	return nil
}

func (s *Schedule) String() string {
	// todo implement me
	return ""
}

func (s *Schedule) parse(raw string, target int) (t uint32, err error) {
	var h, m, sc, ms int
	if raw == "*" {
		switch target {
		case 0:
			return
		default:
			t = uint32(23*msHour + 59*msMin + 59*msSec + 999*msItself)
		}
	}
	if x := reHMSMs.FindStringSubmatch(raw); len(x) > 0 {
		h, _ = strconv.Atoi(strings.TrimLeft(x[1], "0"))
		m, _ = strconv.Atoi(strings.TrimLeft(x[2], "0"))
		sc, _ = strconv.Atoi(strings.TrimLeft(x[3], "0"))
		ms, _ = strconv.Atoi(strings.TrimLeft(x[4], "0"))
	} else if x := reHMS.FindStringSubmatch(raw); len(x) > 0 {
		h, _ = strconv.Atoi(strings.TrimLeft(x[1], "0"))
		m, _ = strconv.Atoi(strings.TrimLeft(x[2], "0"))
		sc, _ = strconv.Atoi(strings.TrimLeft(x[3], "0"))
	} else if x := reHM.FindStringSubmatch(raw); len(x) > 0 {
		h, _ = strconv.Atoi(strings.TrimLeft(x[1], "0"))
		m, _ = strconv.Atoi(strings.TrimLeft(x[2], "0"))
	} else {
		err = ErrSchedBadTime
		return
	}
	if h < 0 || h > 23 {
		err = ErrSchedBadHour
		return
	}
	if m < 0 || m > 59 {
		err = ErrSchedBadMin
		return
	}
	if sc < 0 || sc > 59 {
		err = ErrSchedBadSec
		return
	}
	if ms < 0 || ms > 999 {
		err = ErrSchedBadMsec
		return
	}
	t = uint32(h*msHour + m*msMin + sc*msSec + ms*msItself)
	err = nil
	return
}
