package error_rate

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/koykov/queue"
)

const (
	// Error rate default decay time constant.
	defaultErrorRateTau = time.Minute
)

// ewma error rate implementation based on time-weighted exponentially weighted moving average.
// Decay is lazy: applied on every report and read, without background sweeping.
type ewma struct {
	// math.Float64bits of the current rate. Accessed atomically.
	state uint64
	// UnixNano timestamp of the last update. Accessed atomically.
	ts int64
	// Decay time constant.
	tau time.Duration
	// Clock keeper.
	clk queue.Clock
}

// NewEWMA makes new error rate with given time constant.
func NewEWMA(tau time.Duration, clock queue.Clock) queue.ErrorRater {
	if tau <= 0 {
		tau = defaultErrorRateTau
	}
	if clock == nil {
		clock = queue.NativeClock{}
	}
	return &ewma{
		ts:  clock.Now().UnixNano(),
		tau: tau,
		clk: clock,
	}
}

// Report registers item processing result and folds it into the rate.
func (e *ewma) Report(failed bool) {
	x := 0.0
	if failed {
		x = 1
	}
	tnew := e.clk.Now().UnixNano()
	for {
		old := atomic.LoadUint64(&e.state)
		told := atomic.LoadInt64(&e.ts)
		r := math.Float64frombits(old)
		d := e.decay(tnew - told)
		nr := r*d + (1-d)*x
		if atomic.CompareAndSwapUint64(&e.state, old, math.Float64bits(nr)) {
			atomic.StoreInt64(&e.ts, tnew)
			return
		}
	}
}

// Rate returns current error rate with idle decay applied.
func (e *ewma) Rate() float32 {
	old := atomic.LoadUint64(&e.state)
	if old == 0 {
		return 0
	}
	told := atomic.LoadInt64(&e.ts)
	r := math.Float64frombits(old) * e.decay(e.clk.Now().UnixNano()-told)
	return float32(r)
}

// decay returns e^(-delta/tau), 1 for non-positive delta.
func (e *ewma) decay(delta int64) float64 {
	d := time.Duration(delta)
	if d <= 0 {
		return 1
	}
	return math.Exp(-float64(d) / float64(e.tau))
}
