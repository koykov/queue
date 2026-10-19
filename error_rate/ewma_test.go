package error_rate

import (
	"math"
	"sync"
	"testing"
	"time"
)

// testClock is a controllable queue.Clock for deterministic EWMA tests.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func newTestClock(now time.Time) *testClock { return &testClock{now: now} }

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) add(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

func TestEWMA(t *testing.T) {
	// rateStep is a single scripted action: advance the clock and, optionally,
	// report a processing outcome.
	type rateStep struct {
		advance time.Duration
		report  bool // meaningful only when hasReport is true
		doIt    bool // whether to report on this step
	}

	failStep := func(advance time.Duration) rateStep {
		return rateStep{advance: advance, report: true, doIt: true}
	}

	okStep := func(advance time.Duration) rateStep {
		return rateStep{advance: advance, report: false, doIt: true}
	}

	idleStep := func(advance time.Duration) rateStep {
		return rateStep{advance: advance}
	}

	const tau = 10 * time.Second
	burst := make([]rateStep, 50)
	for i := range burst {
		burst[i] = failStep(tau)
	}

	cases := []struct {
		name  string
		tau   time.Duration
		steps []rateStep
		want  float64
		eps   float64
	}{
		{name: "no_events", tau: tau, want: 0},
		{name: "single_fail", tau: tau, steps: []rateStep{failStep(tau)}, want: 1 - math.Exp(-1)},
		{name: "three_fails", tau: tau, steps: []rateStep{failStep(tau), failStep(tau), failStep(tau)}, want: 1 - math.Exp(-3)},
		{name: "fail_then_ok", tau: tau, steps: []rateStep{failStep(tau), okStep(tau)}, want: math.Exp(-1) - math.Exp(-2)},
		{
			name:  "decay_after_burst",
			tau:   tau,
			steps: append(append([]rateStep{}, burst...), idleStep(tau)),
			want:  math.Exp(-1),
			eps:   1e-6, // burst peak is 1-e^-50, effectively 1
		},
		{
			name:  "ok_after_burst",
			tau:   tau,
			steps: append(append([]rateStep{}, burst...), okStep(tau)),
			want:  math.Exp(-1),
			eps:   1e-6,
		},
		{
			name:  "custom_tau",
			tau:   2 * tau,
			steps: []rateStep{failStep(2 * tau)},
			want:  1 - math.Exp(-1),
		},
		{
			name:  "zero_tau_falls_back_to_default",
			tau:   0,
			steps: []rateStep{failStep(defaultErrorRateTau)},
			want:  1 - math.Exp(-1),
		},
	}

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clk := newTestClock(start)
			e := NewEWMA(c.tau, clk)
			for _, s := range c.steps {
				clk.add(s.advance)
				if s.doIt {
					e.Report(s.report)
				}
			}
			eps := c.eps
			if eps == 0 {
				eps = 1e-6 // Rate() is float32
			}
			if got := float64(e.Rate()); math.Abs(got-c.want) > eps {
				t.Fatalf("Rate() = %v, want %v (eps %v)", got, c.want, eps)
			}
		})
	}
}

func TestEWMAConcurrent(t *testing.T) {
	const tau = time.Second
	clk := newTestClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	e := NewEWMA(tau, clk)

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				clk.add(time.Millisecond)
				e.Report(seed%2 == 0)
			}
		}(g)
	}
	wg.Wait()

	if got := e.Rate(); got < 0 || got > 1 {
		t.Fatalf("Rate() = %v, out of [0,1]", got)
	}
}
