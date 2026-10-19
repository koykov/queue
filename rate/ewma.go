package rate

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/koykov/queue"
)

const (
	// Default decay time constant of the queue error rate EWMA.
	defaultErrorRateTau = time.Minute
)

// ewma is a time-weighted exponentially weighted moving average (EWMA) of
// the queue processing error rate. The value lies in [0,1]: 0 means every recent
// attempt succeeded, 1 means every recent attempt failed.
//
// The rate lazily decays toward zero while no processing occurs, so an idle queue
// reports a healthy (zero) rate. Decay is applied on read and on each report, no
// background sweeping is required.
type ewma struct {
	// state keeps math.Float64bits of the current rate. Accessed atomically.
	state uint64
	// ts keeps UnixNano timestamp of the last update. Accessed atomically.
	ts int64
	// tau is the decay time constant.
	tau time.Duration
	// clk is a clock keeper.
	clk queue.Clock
}

// NewEWMA makes an error rate with given time constant. now is the moment
// the rate starts decaying from, so the very first samples build up gradually
// instead of jumping to the raw outcome.
func NewEWMA(tau time.Duration, clock queue.Clock) queue.Rater {
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

// report folds the outcome of a single processing attempt into the rate.
// failed=true counts an error, failed=false counts a success.
func (e *ewma) Report(status bool) {
	x := 0.0
	if !status {
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

// rate returns the current error rate with lazy idle decay applied.
func (e *ewma) Rate() float32 {
	old := atomic.LoadUint64(&e.state)
	if old == 0 {
		return 0
	}
	told := atomic.LoadInt64(&e.ts)
	r := math.Float64frombits(old) * e.decay(e.clk.Now().UnixNano()-told)
	return float32(r)
}

// decay returns e^(-delta/tau) clamped to 1 for non-positive deltas.
func (e *ewma) decay(delta int64) float64 {
	d := time.Duration(delta)
	if d <= 0 {
		return 1
	}
	return math.Exp(-float64(d) / float64(e.tau))
}
