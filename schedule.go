package blqueue

import (
	"bytes"
	"fmt"
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
	buf  []schedRule
	once sync.Once
}

type schedRule struct {
	lt, rt uint32
	wn, wx uint32
	wf, sf float32
}

var (
	reHMSMs = regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2})\.(\d{3})`)
	reHMS   = regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2})`)
	reHM    = regexp.MustCompile(`(\d{2}):(\d{2})`)
)

func NewSchedule() *Schedule {
	s := &Schedule{}
	return s
}

func (s *Schedule) init() {
	// todo implement me
}

func (s *Schedule) AddRange(raw string, min, max uint32, wakeup, sleep float32) (err error) {
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
	s.buf = append(s.buf, schedRule{
		lt: lt,
		rt: rt,
		wn: min,
		wx: max,
		wf: wakeup,
		sf: sleep,
	})
	return nil
}

func (s *Schedule) String() string {
	var buf bytes.Buffer
	_, _ = buf.WriteString("[\n")
	for i := 0; i < len(s.buf); i++ {
		r := s.buf[i]
		_ = buf.WriteByte('\t')
		_, _ = buf.WriteString(s.fmtTime(r.lt))
		_ = buf.WriteByte('-')
		_, _ = buf.WriteString(s.fmtTime(r.rt))
		_, _ = buf.WriteString(" min: ")
		_, _ = buf.WriteString(strconv.Itoa(int(r.wn)))
		_, _ = buf.WriteString(" max: ")
		_, _ = buf.WriteString(strconv.Itoa(int(r.wx)))
		_, _ = buf.WriteString(" wakeup: ")
		_, _ = buf.WriteString(strconv.FormatFloat(float64(r.wf), 'f', -1, 32))
		_, _ = buf.WriteString(" sleep: ")
		_, _ = buf.WriteString(strconv.FormatFloat(float64(r.sf), 'f', -1, 32))
		if i > 0 {
			_ = buf.WriteByte(',')
		}
		_ = buf.WriteByte('\n')
	}
	_ = buf.WriteByte(']')
	return buf.String()
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

func (s *Schedule) fmtTime(t uint32) string {
	h := t / msHour
	t = t % msHour
	m := t / msMin
	t = t % msMin
	sc := t / msSec
	t = t % msSec
	ms := t
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, sc, ms)
}
