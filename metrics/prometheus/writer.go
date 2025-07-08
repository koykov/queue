package prometheus

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsWriter is a Prometheus implementation of queue.MetricsWriter.
type MetricsWriter struct {
	name string
	prec time.Duration
}

var (
	promQueueSize, promSubqSize, promWorkerIdle, promWorkerActive, promWorkerSleep *prometheus.GaugeVec
	promQueueIn, promQueueOut, promQueueRetry, promQueueLeak, promQueueDeadline, promQueueLost,
	promSubqIn, promSubqOut, promSubqLeak *prometheus.CounterVec

	promWorkerWait *prometheus.HistogramVec
	promRetryDelay *prometheus.HistogramVec

	_ = NewPrometheusMetrics
)

func init() {
	promWorkerIdle = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_workers_idle",
		Help: "Indicates how many workers idle.",
	}, []string{"queue"})
	promWorkerActive = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_workers_active",
		Help: "Indicates how many workers active.",
	}, []string{"queue"})
	promWorkerSleep = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_workers_sleep",
		Help: "Indicates how many workers sleep.",
	}, []string{"queue"})

	promQueueSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_size",
		Help: "Actual queue size.",
	}, []string{"queue"})

	promQueueIn = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_in",
		Help: "How many items comes to the queue.",
	}, []string{"queue"})
	promQueueOut = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_out",
		Help: "How many items leaves queue.",
	}, []string{"queue"})
	promQueueRetry = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_retry",
		Help: "How many retries occurs.",
	}, []string{"queue"})
	promQueueLeak = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_leak",
		Help: "How many items dropped on the floor due to queue is full.",
	}, []string{"queue", "dir"})
	promQueueDeadline = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_deadline",
		Help: "How many processing skips due to deadline.",
	}, []string{"queue"})
	promQueueLost = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_lost",
		Help: "How many items throw to the trash due to force close.",
	}, []string{"queue"})

	buckets := append(prometheus.DefBuckets, []float64{15, 20, 30, 40, 50, 100, 150, 200, 250, 500, 1000, 1500, 2000, 3000, 5000}...)
	promWorkerWait = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "queue_wait",
		Help:    "How long worker waits due to delayed execution.",
		Buckets: buckets,
	}, []string{"queue"})
	promRetryDelay = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "queue_retry_delay",
		Help:    "How long worker waits between retry attempts.",
		Buckets: buckets,
	}, []string{"queue"})

	promSubqSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_subq_size",
		Help: "Actual queue size.",
	}, []string{"queue", "subq"})
	promSubqIn = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_subq_in",
		Help: "How many items comes to the sub-queue.",
	}, []string{"queue", "subq"})
	promSubqOut = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_subq_out",
		Help: "How many items leaves sub-queue.",
	}, []string{"queue", "subq"})
	promSubqLeak = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_subq_leak",
		Help: "How many items dropped on the floor due to sub-queue is full.",
	}, []string{"queue", "subq"})

	prometheus.MustRegister(promWorkerIdle, promWorkerActive, promWorkerSleep, promQueueSize,
		promQueueIn, promQueueOut, promQueueRetry, promQueueLeak, promQueueLost, promQueueDeadline,
		promWorkerWait,
		promSubqSize, promSubqIn, promSubqOut, promSubqLeak, promRetryDelay)
}

func NewPrometheusMetrics(name string) *MetricsWriter {
	return NewPrometheusMetricsWP(name, time.Nanosecond)
}

func NewPrometheusMetricsWP(name string, precision time.Duration) *MetricsWriter {
	if precision == 0 {
		precision = time.Nanosecond
	}
	m := &MetricsWriter{
		name: name,
		prec: precision,
	}
	return m
}

func (w MetricsWriter) WorkerSetup(active, sleep, stop uint) {
	promWorkerActive.DeleteLabelValues(w.name)
	promWorkerSleep.DeleteLabelValues(w.name)
	promWorkerIdle.DeleteLabelValues(w.name)

	promWorkerActive.WithLabelValues(w.name).Add(float64(active))
	promWorkerSleep.WithLabelValues(w.name).Add(float64(sleep))
	promWorkerIdle.WithLabelValues(w.name).Add(float64(stop))
}

func (w MetricsWriter) WorkerInit(_ uint32) {
	promWorkerActive.WithLabelValues(w.name).Inc()
	promWorkerIdle.WithLabelValues(w.name).Add(-1)
}

func (w MetricsWriter) WorkerSleep(_ uint32) {
	promWorkerSleep.WithLabelValues(w.name).Inc()
	promWorkerActive.WithLabelValues(w.name).Add(-1)
}

func (w MetricsWriter) WorkerWakeup(_ uint32) {
	promWorkerActive.WithLabelValues(w.name).Inc()
	promWorkerSleep.WithLabelValues(w.name).Add(-1)
}

func (w MetricsWriter) WorkerWait(_ uint32, delay time.Duration) {
	promWorkerWait.WithLabelValues(w.name).Observe(float64(delay.Nanoseconds() / int64(w.prec)))
}

func (w MetricsWriter) WorkerStop(_ uint32, force bool, status string) {
	promWorkerIdle.WithLabelValues(w.name).Inc()
	if force {
		switch status {
		case "active":
			promWorkerActive.WithLabelValues(w.name).Add(-1)
		case "sleep":
			promWorkerSleep.WithLabelValues(w.name).Add(-1)
		}
	} else {
		promWorkerSleep.WithLabelValues(w.name).Add(-1)
	}
}

func (w MetricsWriter) QueuePut() {
	promQueueIn.WithLabelValues(w.name).Inc()
	promQueueSize.WithLabelValues(w.name).Inc()
}

func (w MetricsWriter) QueuePull() {
	promQueueOut.WithLabelValues(w.name).Inc()
	promQueueSize.WithLabelValues(w.name).Dec()
}

func (w MetricsWriter) QueueRetry(delay time.Duration) {
	promQueueRetry.WithLabelValues(w.name).Inc()
	promRetryDelay.WithLabelValues(w.name).Observe(float64(delay.Nanoseconds() / int64(w.prec)))
}

func (w MetricsWriter) QueueLeak(direction string) {
	promQueueLeak.WithLabelValues(w.name, direction).Inc()
	promQueueSize.WithLabelValues(w.name).Dec()
}

func (w MetricsWriter) QueueDeadline() {
	promQueueDeadline.WithLabelValues(w.name).Inc()
	promQueueSize.WithLabelValues(w.name).Dec()
}

func (w MetricsWriter) QueueLost() {
	promQueueLost.WithLabelValues(w.name).Inc()
	promQueueSize.WithLabelValues(w.name).Dec()
}

func (w MetricsWriter) SubqPut(subq string) {
	promSubqIn.WithLabelValues(w.name, subq).Inc()
	promSubqSize.WithLabelValues(w.name, subq).Inc()
}

func (w MetricsWriter) SubqPull(subq string) {
	promSubqOut.WithLabelValues(w.name, subq).Inc()
	promSubqSize.WithLabelValues(w.name, subq).Dec()
}

func (w MetricsWriter) SubqLeak(subq string) {
	promSubqLeak.WithLabelValues(w.name, subq).Inc()
	promSubqSize.WithLabelValues(w.name, subq).Dec()
}
