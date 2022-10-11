package queue

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	msItself = 1
	msSec    = 1000 * msItself
	msMin    = 60 * msSec
	msHour   = 60 * msMin
)

// Schedule describes time ranges with specific queue params.
type Schedule struct {
	buf []schedRule
	srt bool
}

type schedRule struct {
	// Left and right daily timestamps.
	lt, rt uint32
	// Params to use between lt and rt.
	params ScheduleParams
}

// ScheduleParams describes queue params for specific time range.
type ScheduleParams realtimeParams

var (
	reHMSMs = regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2})\.(\d{3})`)
	reHMS   = regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2})`)
	reHM    = regexp.MustCompile(`(\d{2}):(\d{2})`)
)

// NewSchedule makes new schedule instance.
// Don't share it among many queues.
func NewSchedule() *Schedule {
	s := &Schedule{}
	return s
}

// AddRange registers specific params for given time range.
// Raw specifies time range in format `<left time>-<right time>`. Time point may be in three formats:
// * HH:MM
// * HH:MM:SS
// * HH:MM:SS.MSC (msc is a millisecond 0-999).
// All time ranges outside registered will use default params specified in config (WorkersMin, WorkersMax, WakeupFactor
// and SleepFactor).
func (s *Schedule) AddRange(raw string, params ScheduleParams) (err error) {
	if params.WorkersMax == 0 {
		return ErrSchedZeroMax
	}
	if params.WorkersMin > params.WorkersMax {
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
	s.srt = false
	s.buf = append(s.buf, schedRule{
		lt: lt, rt: rt,
		params: params,
	})
	return nil
}

// Get returns queue params if current time hits to the one of registered ranges.
// Param schedID indicates which time range hits and contains -1 on miss.
func (s *Schedule) Get() (params ScheduleParams, schedID int) {
	l := len(s.buf)
	if l == 0 {
		return
	}
	schedID = -1
	s.sort()
	now := time.Now()
	h, m, sc, ms := now.Hour(), now.Minute(), now.Second(), now.Nanosecond()/1e6
	t := uint32(h*msHour + m*msMin + sc*msSec + ms)
	_ = s.buf[l-1]
	for i := 0; i < l; i++ {
		r := s.buf[i]
		if r.lt <= t && r.rt > t {
			params, schedID = r.params, i
			return
		}
	}
	return
}

// Parse and convert time point to daily timestamp.
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

// WorkersMaxDaily returns maximum number of workers from registered time ranges.
func (s *Schedule) WorkersMaxDaily() (max uint32) {
	l := len(s.buf)
	if l == 0 {
		return 0
	}
	_ = s.buf[l-1]
	for i := 0; i < l; i++ {
		if s.buf[i].params.WorkersMax > max {
			max = s.buf[i].params.WorkersMax
		}
	}
	return
}

func (s *Schedule) String() string {
	s.sort()
	var buf bytes.Buffer
	_, _ = buf.WriteString("[\n")
	for i := 0; i < len(s.buf); i++ {
		r := s.buf[i]
		_ = buf.WriteByte('\t')
		_, _ = buf.WriteString(s.fmtTime(r.lt))
		_ = buf.WriteByte('-')
		_, _ = buf.WriteString(s.fmtTime(r.rt))
		_, _ = buf.WriteString(" min: ")
		_, _ = buf.WriteString(strconv.Itoa(int(r.params.WorkersMin)))
		_, _ = buf.WriteString(" max: ")
		_, _ = buf.WriteString(strconv.Itoa(int(r.params.WorkersMax)))
		_, _ = buf.WriteString(" wakeup: ")
		_, _ = buf.WriteString(strconv.FormatFloat(float64(r.params.WakeupFactor), 'f', -1, 32))
		_, _ = buf.WriteString(" sleep: ")
		_, _ = buf.WriteString(strconv.FormatFloat(float64(r.params.SleepFactor), 'f', -1, 32))
		if i > 0 {
			_ = buf.WriteByte(',')
		}
		_ = buf.WriteByte('\n')
	}
	_ = buf.WriteByte(']')
	return buf.String()
}

// Format daily timestamp to time point.
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

// Copy copies schedule instance to protect queue from changing params after start.
// It means that after starting queue all schedule modifications will have no effect.
func (s *Schedule) Copy() *Schedule {
	s.sort()
	cpy := &Schedule{}
	cpy.buf = append(cpy.buf, s.buf...)
	return cpy
}

func (s *Schedule) sort() {
	if s.srt == true {
		return
	}
	s.srt = true
	sort.Sort(s)
}

func (s *Schedule) Len() int {
	return len(s.buf)
}

func (s *Schedule) Less(i, j int) bool {
	return s.buf[i].lt < s.buf[j].lt
}

func (s *Schedule) Swap(i, j int) {
	s.buf[i], s.buf[j] = s.buf[j], s.buf[i]
}
