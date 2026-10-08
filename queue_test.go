package queue

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/koykov/queue/qos"
)

// testWorker is a configurable mock worker used across the tests.
type testWorker struct {
	gate      chan struct{} // if set, the first Do blocks until the channel is closed
	started   chan struct{} // if set, closed on the first Do
	startOnce sync.Once
	delay     time.Duration // processing delay
	fail      error         // if set, Do returns this error
	failLimit int32         // fail only the first failLimit attempts (0 means always fail)

	mu       sync.Mutex
	got      []any
	firstAt  time.Time
	attempts int32
}

func (w *testWorker) Do(x any) error {
	if w.started != nil {
		w.startOnce.Do(func() { close(w.started) })
	}
	n := atomic.AddInt32(&w.attempts, 1)
	w.mu.Lock()
	if w.firstAt.IsZero() {
		w.firstAt = time.Now()
	}
	w.mu.Unlock()

	if w.gate != nil && n == 1 {
		<-w.gate
	}
	if w.delay > 0 {
		time.Sleep(w.delay)
	}
	if w.fail != nil && (w.failLimit == 0 || n <= w.failLimit) {
		return w.fail
	}

	w.mu.Lock()
	w.got = append(w.got, x)
	w.mu.Unlock()
	return nil
}

func (w *testWorker) processed() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.got)
}

func (w *testWorker) attemptsN() int { return int(atomic.LoadInt32(&w.attempts)) }

func (w *testWorker) firstProcessedAt() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.firstAt
}

// testDLQ is a mock DLQ that records every leaked payload.
type testDLQ struct {
	mu  sync.Mutex
	got []any
}

func (d *testDLQ) Enqueue(x any) error {
	d.mu.Lock()
	d.got = append(d.got, x)
	d.mu.Unlock()
	return nil
}

func (d *testDLQ) len() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.got)
}

func waitCond(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition was not met in time")
}

func TestNew(t *testing.T) {
	cases := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{name: "nil_config", config: nil, wantErr: ErrNoConfig},
		{name: "no_capacity", config: &Config{Worker: &testWorker{}}, wantErr: ErrNoCapacity},
		{name: "no_worker", config: &Config{Capacity: 4}, wantErr: ErrNoWorker},
		{name: "no_workers", config: &Config{Capacity: 4, Worker: &testWorker{}}, wantErr: ErrNoWorkers},
		{
			name: "qos_no_evaluator",
			config: &Config{
				Worker:  &testWorker{},
				Workers: 1,
				QoS:     qos.New(qos.RR, nil).AddQueue(qos.Queue{Name: "a", Capacity: 8, Weight: 1}),
				// second queue is required by Validate, but evaluator check happens first
			},
			wantErr: qos.ErrNoEvaluator,
		},
		{name: "valid", config: &Config{Capacity: 4, Workers: 1, Worker: &testWorker{}}, wantErr: nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q, err := New(c.config)
			if err != c.wantErr {
				t.Fatalf("New() error = %v, want %v", err, c.wantErr)
			}
			// A queue that failed to init has no engine, so it must not be closed.
			if err == nil && q != nil {
				_ = q.Close()
			}
		})
	}
}

func TestAccessors(t *testing.T) {
	cases := []struct {
		name         string
		config       *Config
		wantCapacity int
	}{
		{
			name:         "fifo",
			config:       &Config{Capacity: 10, Workers: 1, Worker: &testWorker{}},
			wantCapacity: 10,
		},
		{
			name:         "streams",
			config:       &Config{Capacity: 20, Streams: 4, Workers: 1, Worker: &testWorker{}},
			wantCapacity: 20,
		},
		{
			name: "qos",
			config: &Config{
				Worker:  &testWorker{},
				Workers: 1,
				QoS: qos.New(qos.RR, qos.DummyPriorityEvaluator{}).
					SetEgressCapacity(16).
					AddQueue(qos.Queue{Name: "a", Capacity: 10, Weight: 1}).
					AddQueue(qos.Queue{Name: "b", Capacity: 20, Weight: 1}),
			},
			wantCapacity: 46,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q, err := New(c.config)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = q.Close() }()

			if got := q.Capacity(); got != c.wantCapacity {
				t.Errorf("Capacity() = %d, want %d", got, c.wantCapacity)
			}
			if got := q.Size(); got != 0 {
				t.Errorf("Size() = %d, want 0", got)
			}
			if got := q.Rate(); got != 0 {
				t.Errorf("Rate() = %f, want 0", got)
			}
		})
	}
}

func TestProcess(t *testing.T) {
	someErr := errors.New("boom")
	cases := []struct {
		name          string
		workers       uint32
		streams       uint32
		capacity      uint64
		items         int
		useQoS        bool
		cfgDelay      time.Duration
		jobDeadline   time.Duration
		maxRetries    uint32
		failLimit     int32
		fail          bool
		failToDLQ     bool
		deadlineToDLQ bool
		useDLQ        bool
		wantProcessed int
		wantAttempts  int
		wantDLQ       int
		minFirstDelay time.Duration
		settle        bool
	}{
		{name: "static_single", workers: 1, capacity: 8, items: 5, wantProcessed: 5},
		{name: "streams", streams: 4, capacity: 16, items: 16, wantProcessed: 16},
		{name: "qos", useQoS: true, items: 10, wantProcessed: 10},
		{name: "delayed", workers: 1, capacity: 8, items: 1, cfgDelay: 50 * time.Millisecond, wantProcessed: 1, minFirstDelay: 40 * time.Millisecond},
		{name: "deadline_skip", workers: 1, capacity: 8, items: 1, jobDeadline: time.Nanosecond, wantProcessed: 0, settle: true},
		{name: "deadline_to_dlq", workers: 1, capacity: 8, items: 1, jobDeadline: time.Nanosecond, useDLQ: true, deadlineToDLQ: true, wantProcessed: 0, wantDLQ: 1},
		{name: "retry_then_success", workers: 1, capacity: 8, items: 1, maxRetries: 2, fail: true, failLimit: 1, wantProcessed: 1, wantAttempts: 2},
		{name: "retries_exhausted", workers: 1, capacity: 8, items: 1, maxRetries: 2, fail: true, wantProcessed: 0, wantAttempts: 3},
		{name: "fail_to_dlq", workers: 1, capacity: 8, items: 1, maxRetries: 1, fail: true, failToDLQ: true, useDLQ: true, wantProcessed: 0, wantAttempts: 2, wantDLQ: 1},
		{name: "fail_dropped", workers: 1, capacity: 8, items: 1, maxRetries: 0, fail: true, wantProcessed: 0, wantAttempts: 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &testWorker{failLimit: c.failLimit}
			if c.fail {
				w.fail = someErr
			}
			dlq := &testDLQ{}

			cfg := &Config{Capacity: c.capacity, Worker: w, MaxRetries: c.maxRetries, DelayInterval: c.cfgDelay}
			if c.workers > 0 {
				cfg.Workers = c.workers
			} else if !c.useQoS {
				cfg.Workers = 1
			}
			cfg.Streams = c.streams
			if c.useDLQ {
				cfg.DLQ = dlq
				cfg.FailToDLQ = c.failToDLQ
				cfg.DeadlineToDLQ = c.deadlineToDLQ
			}
			if c.useQoS {
				cfg.Workers = 1
				cfg.QoS = qos.New(qos.RR, qos.DummyPriorityEvaluator{}).
					SetEgressCapacity(32).
					AddQueue(qos.Queue{Name: "a", Capacity: 16, Weight: 1}).
					AddQueue(qos.Queue{Name: "b", Capacity: 16, Weight: 1})
			}

			q, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = q.Close() }()

			start := time.Now()
			for i := 0; i < c.items; i++ {
				x := any(i)
				if c.jobDeadline > 0 {
					x = Job{Payload: i, DeadlineInterval: c.jobDeadline}
				}
				if err := q.Enqueue(x); err != nil {
					t.Fatalf("Enqueue: %v", err)
				}
			}

			switch {
			case c.wantProcessed > 0:
				waitCond(t, func() bool { return w.processed() >= c.wantProcessed })
			case c.wantDLQ > 0:
				waitCond(t, func() bool { return dlq.len() >= c.wantDLQ })
			case c.wantAttempts > 0:
				waitCond(t, func() bool { return w.attemptsN() >= c.wantAttempts })
			case c.settle:
				time.Sleep(150 * time.Millisecond)
			}

			if got := w.processed(); got != c.wantProcessed {
				t.Errorf("processed = %d, want %d", got, c.wantProcessed)
			}
			if c.wantAttempts > 0 {
				if got := w.attemptsN(); got != c.wantAttempts {
					t.Errorf("attempts = %d, want %d", got, c.wantAttempts)
				}
			}
			if got := dlq.len(); got != c.wantDLQ {
				t.Errorf("DLQ = %d, want %d", got, c.wantDLQ)
			}
			if c.minFirstDelay > 0 {
				if d := w.firstProcessedAt().Sub(start); d < c.minFirstDelay {
					t.Errorf("processing started after %v, want >= %v", d, c.minFirstDelay)
				}
			}
		})
	}
}

func TestDLQLeak(t *testing.T) {
	cases := []struct {
		name     string
		capacity uint64
		extra    int
		leakDir  LeakDirection
		wantDLQ  int
	}{
		{name: "rear", capacity: 4, extra: 6, leakDir: LeakDirectionRear, wantDLQ: 2},
		{name: "front", capacity: 4, extra: 6, leakDir: LeakDirectionFront, wantDLQ: 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &testWorker{gate: make(chan struct{})}
			dlq := &testDLQ{}
			q, err := New(&Config{
				Capacity:      c.capacity,
				Workers:       1,
				Worker:        w,
				DLQ:           dlq,
				LeakDirection: c.leakDir,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { close(w.gate); _ = q.Close() }()

			// Occupy the worker so the channel fills up.
			if err := q.Enqueue(-1); err != nil {
				t.Fatal(err)
			}
			waitCond(t, func() bool { return w.attemptsN() >= 1 })

			// Fill the channel and overflow: the extra items must leak to DLQ.
			for i := 0; i < c.extra; i++ {
				if err := q.Enqueue(i); err != nil {
					t.Fatalf("Enqueue: %v", err)
				}
			}
			waitCond(t, func() bool { return dlq.len() >= c.wantDLQ })
			if got := dlq.len(); got != c.wantDLQ {
				t.Errorf("DLQ = %d, want %d", got, c.wantDLQ)
			}
		})
	}
}

func TestClose(t *testing.T) {
	cases := []struct {
		name          string
		workers       uint32
		workersMin    uint32
		workersMax    uint32
		capacity      uint64
		items         int
		gate          bool
		doDelay       time.Duration
		force         bool
		sync          bool
		useDLQ        bool
		waitInflight  bool
		releaseAfter  time.Duration
		wantProcessed int
		waitProcessed bool
		wantDLQ       int
		minElapsed    time.Duration
		maxElapsed    time.Duration
	}{
		{
			name: "graceful_sync_drains", workers: 1, capacity: 16, items: 8, sync: true,
			wantProcessed: 8, maxElapsed: 3 * time.Second,
		},
		{
			name: "graceful_sync_balanced_drains", workersMin: 1, workersMax: 4, capacity: 64, items: 20,
			sync: true, wantProcessed: 20, maxElapsed: 5 * time.Second,
		},
		{
			name: "graceful_sync_drains_blocked_first", workers: 1, capacity: 16, items: 8, gate: true,
			releaseAfter: 100 * time.Millisecond, sync: true, wantProcessed: 8, maxElapsed: 3 * time.Second,
		},
		{
			name: "force_sync_waits_inflight", workers: 1, capacity: 4, items: 1, doDelay: 300 * time.Millisecond,
			force: true, sync: true, waitInflight: true, wantProcessed: 1,
			minElapsed: 250 * time.Millisecond, maxElapsed: 3 * time.Second,
		},
		{
			name: "graceful_async_returns_early", workers: 1, capacity: 4, items: 1, doDelay: 300 * time.Millisecond,
			waitInflight: true, wantProcessed: 1, waitProcessed: true, maxElapsed: 150 * time.Millisecond,
		},
		{
			name: "force_async_returns_early", workers: 1, capacity: 4, items: 1, doDelay: 300 * time.Millisecond,
			force: true, waitInflight: true, wantProcessed: 1, waitProcessed: true, maxElapsed: 150 * time.Millisecond,
		},
		{
			name: "force_async_leaks_to_dlq", workers: 1, capacity: 16, items: 4, gate: true,
			force: true, useDLQ: true, waitInflight: true, releaseAfter: 200 * time.Millisecond,
			wantProcessed: 1, waitProcessed: true, wantDLQ: 3, maxElapsed: time.Second,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &testWorker{delay: c.doDelay}
			if c.gate {
				w.gate = make(chan struct{})
			}
			if c.waitInflight {
				w.started = make(chan struct{})
			}
			dlq := &testDLQ{}
			cfg := &Config{Capacity: c.capacity, Worker: w, CloseStrategy: CloseStrategyAsynchronous}
			if c.workers > 0 {
				cfg.Workers = c.workers
			} else {
				cfg.WorkersMin = c.workersMin
				cfg.WorkersMax = c.workersMax
			}
			if c.sync {
				cfg.CloseStrategy = CloseStrategySynchronous
			}
			if c.useDLQ {
				cfg.DLQ = dlq
			}
			q, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < c.items; i++ {
				if err := q.Enqueue(i); err != nil {
					t.Fatal(err)
				}
			}
			if c.waitInflight {
				select {
				case <-w.started:
				case <-time.After(time.Second):
					t.Fatal("worker did not start processing")
				}
			}

			start := time.Now()
			ret := make(chan error, 1)
			go func() {
				if c.force {
					ret <- q.ForceClose()
				} else {
					ret <- q.Close()
				}
			}()
			if c.releaseAfter > 0 {
				time.Sleep(c.releaseAfter)
				close(w.gate)
			}
			select {
			case err := <-ret:
				if err != nil {
					t.Fatalf("close error: %v", err)
				}
			case <-time.After(4 * time.Second):
				t.Fatal("HANG: close did not return")
			}
			elapsed := time.Since(start)

			if c.minElapsed > 0 && elapsed < c.minElapsed {
				t.Errorf("close returned too early: %v < %v", elapsed, c.minElapsed)
			}
			if c.maxElapsed > 0 && elapsed > c.maxElapsed {
				t.Errorf("close returned too late: %v > %v", elapsed, c.maxElapsed)
			}
			if c.waitProcessed {
				waitCond(t, func() bool { return w.processed() >= c.wantProcessed })
			}
			if got := w.processed(); got != c.wantProcessed {
				t.Errorf("processed = %d, want %d", got, c.wantProcessed)
			}
			if got := dlq.len(); got != c.wantDLQ {
				t.Errorf("DLQ = %d, want %d", got, c.wantDLQ)
			}
		})
	}
}

func TestCloseStopsSleepers(t *testing.T) {
	cases := []struct {
		name  string
		force bool
		sync  bool
	}{
		{name: "graceful_sync", force: false, sync: true},
		{name: "force_sync", force: true, sync: true},
		{name: "graceful_async", force: false, sync: false},
		{name: "force_async", force: true, sync: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q, err := New(&Config{
				Capacity:      4,
				WorkersMin:    1,
				WorkersMax:    4,
				Worker:        &testWorker{},
				CloseStrategy: CloseStrategyAsynchronous,
			})
			if err != nil {
				t.Fatal(err)
			}
			if c.sync {
				q.config.CloseStrategy = CloseStrategySynchronous
			}

			// Emulate three sleeping workers: live goroutines blocked on ctl.
			var ws []*worker
			for i := 1; i <= 3; i++ {
				w := q.workers[i]
				w.setStatus(WorkerStatusSleep)
				ws = append(ws, w)
				go w.await(q)
			}

			if c.force {
				_ = q.ForceClose()
			} else {
				_ = q.Close()
			}
			// A stopped sleeper ends up idle. The eol slot is one-shot and may already be consumed
			// by a synchronous close, so assert on the worker status instead.
			deadline := time.Now().Add(2 * time.Second)
			for i, w := range ws {
				for w.getStatus() != WorkerStatusIdle {
					if time.Now().After(deadline) {
						t.Fatalf("sleeping worker %d was not stopped", i)
					}
					runtime.Gosched()
					time.Sleep(time.Millisecond)
				}
			}
		})
	}
}

func TestEnqueueAfterClose(t *testing.T) {
	cases := []struct {
		name  string
		force bool
		sync  bool
	}{
		{name: "async_close", force: false, sync: false},
		{name: "async_force_close", force: true, sync: false},
		{name: "sync_close", force: false, sync: true},
		{name: "sync_force_close", force: true, sync: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &Config{Capacity: 4, Workers: 1, Worker: &testWorker{}}
			if c.sync {
				cfg.CloseStrategy = CloseStrategySynchronous
			}
			q, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if c.force {
				_ = q.ForceClose()
			} else {
				_ = q.Close()
			}
			if err := q.Enqueue(1); err != ErrQueueClosed {
				t.Errorf("Enqueue after close = %v, want %v", err, ErrQueueClosed)
			}
			// Second close must report the queue is already closed.
			var err2 error
			if c.force {
				err2 = q.ForceClose()
			} else {
				err2 = q.Close()
			}
			if err2 != ErrQueueClosed {
				t.Errorf("second close = %v, want %v", err2, ErrQueueClosed)
			}
		})
	}
}

func TestWorkerRestartGate(t *testing.T) {
	cases := []struct {
		name       string
		running    int32
		wantActive bool
	}{
		{name: "previous_life_finished", running: 0, wantActive: true},
		{name: "previous_life_in_flight", running: 1, wantActive: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := &testWorker{gate: make(chan struct{}), started: make(chan struct{})}
			q, err := New(&Config{
				Capacity:          8,
				WorkersMin:        1,
				WorkersMax:        2,
				WakeupFactor:      .5,
				SleepFactor:       .1,
				HeartbeatInterval: time.Minute,
				Worker:            g,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { close(g.gate); _ = q.Close() }()

			// Fill the queue: worker #0 blocks on the first item, so rate stays high.
			for i := 0; i < 8; i++ {
				_ = q.Enqueue(i)
			}
			select {
			case <-g.started:
			case <-time.After(time.Second):
				t.Fatal("worker did not start")
			}

			w1 := q.workers[1]
			w1.setStatus(WorkerStatusIdle)
			atomic.StoreInt32(&w1.running, c.running)

			q.calibrate(true)

			gotActive := w1.getStatus() == WorkerStatusActive
			if gotActive != c.wantActive {
				t.Fatalf("worker started=%v, want %v (running=%d)", gotActive, c.wantActive, c.running)
			}
		})
	}
}

func TestConcurrentEnqueue(t *testing.T) {
	cases := []struct {
		name       string
		goroutines int
		perG       int
		capacity   uint64
	}{
		{name: "8x100", goroutines: 8, perG: 100, capacity: 1024},
		{name: "4x250", goroutines: 4, perG: 250, capacity: 1024},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &testWorker{}
			q, err := New(&Config{Capacity: c.capacity, Workers: 2, Worker: w})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = q.Close() }()

			var wg sync.WaitGroup
			for g := 0; g < c.goroutines; g++ {
				wg.Add(1)
				go func(base int) {
					defer wg.Done()
					for i := 0; i < c.perG; i++ {
						if err := q.Enqueue(base*c.perG + i); err != nil {
							t.Errorf("Enqueue: %v", err)
							return
						}
					}
				}(g)
			}
			wg.Wait()

			total := c.goroutines * c.perG
			waitCond(t, func() bool { return w.processed() >= total })
			if got := w.processed(); got != total {
				t.Errorf("processed = %d, want %d", got, total)
			}
		})
	}
}
