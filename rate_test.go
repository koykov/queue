package queue

import (
	"errors"
	"sync"
	"testing"
)

// fakeRater records every report and returns a fixed rate.
type fakeRater struct {
	mu      sync.Mutex
	reports []bool
	rate    float32
}

func (r *fakeRater) Report(failed bool) {
	r.mu.Lock()
	r.reports = append(r.reports, failed)
	r.mu.Unlock()
}

func (r *fakeRater) Rate() float32 { return r.rate }

func (r *fakeRater) snapshot() []bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]bool(nil), r.reports...)
}

func TestInterfaceHasErrorRate(_ *testing.T) {
	// Method expression: compiles only if Interface declares ErrorRate, and does
	// not invoke it (so no nil receiver is dereferenced).
	var _ = Interface.ErrorRate
}

func TestQueueErrorRateDelegates(t *testing.T) {
	cases := []struct {
		name  string
		rater ErrorRater
		want  float32
	}{
		{name: "default_dummy", rater: nil, want: 0},
		{name: "custom", rater: &fakeRater{rate: 0.42}, want: 0.42},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q, err := New(&Config{Capacity: 4, Workers: 1, Worker: &testWorker{}, ErrorRate: c.rater})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = q.Close() }()

			if got := q.ErrorRate(); got != c.want {
				t.Fatalf("ErrorRate() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestQueueErrorRateReport(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name       string
		fail       error
		items      int
		wantFailed bool
	}{
		{name: "success", items: 4, wantFailed: false},
		{name: "failure", fail: boom, items: 4, wantFailed: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &fakeRater{}
			w := &testWorker{fail: c.fail}
			q, err := New(&Config{Capacity: uint64(c.items), Workers: 1, Worker: w, ErrorRate: r})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = q.Close() }()

			for i := 0; i < c.items; i++ {
				_ = q.Enqueue(i)
			}
			waitCond(t, func() bool { return len(r.snapshot()) == c.items })

			for i, got := range r.snapshot() {
				if got != c.wantFailed {
					t.Fatalf("report #%d failed = %v, want %v", i, got, c.wantFailed)
				}
			}
		})
	}
}
