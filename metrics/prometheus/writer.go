package prometheus

import "github.com/prometheus/client_golang/prometheus"

type Prometheus struct {
	queue string
}

var (
	queueSize *prometheus.GaugeVec

	workerIdle, workerActive, workerSleep,
	queueIn, queueOut, queueLeak *prometheus.CounterVec
)

func init() {
	workerIdle = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_workers_idle",
		Help: "Indicates how many workers idle.",
	}, []string{"queue"})
	workerActive = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_workers_active",
		Help: "Indicates how many workers active.",
	}, []string{"queue"})
	workerSleep = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_workers_sleep",
		Help: "Indicates how many workers sleep.",
	}, []string{"queue"})

	queueSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_size",
		Help: "Actual queue size.",
	}, []string{"queue"})

	queueIn = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_in",
		Help: "How many items comes to the queue.",
	}, []string{"queue"})
	queueOut = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_out",
		Help: "How many items leaves queue.",
	}, []string{"queue"})
	queueLeak = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_leak",
		Help: "How many items dropped on the floor due to queue is full.",
	}, []string{"queue"})

	prometheus.MustRegister(workerIdle, workerActive, workerSleep, queueSize, queueIn, queueOut, queueLeak)
}

func NewMetricsWriter(queueKey string) *Prometheus {
	m := &Prometheus{queue: queueKey}
	return m
}

func (m *Prometheus) WorkerSleep(_ uint32) {
	workerSleep.WithLabelValues(m.queue).Inc()
	workerActive.WithLabelValues(m.queue).Add(-1)
}

func (m *Prometheus) WorkerWakeup(_ uint32) {
	workerActive.WithLabelValues(m.queue).Inc()
	workerSleep.WithLabelValues(m.queue).Add(-1)
}

func (m *Prometheus) WorkerStop(_ uint32) {
	workerIdle.WithLabelValues(m.queue).Inc()
	workerActive.WithLabelValues(m.queue).Add(-1)
}

func (m *Prometheus) QueuePut() {
	queueIn.WithLabelValues(m.queue).Inc()
	queueSize.WithLabelValues(m.queue).Inc()
}

func (m *Prometheus) QueuePull() {
	queueOut.WithLabelValues(m.queue).Inc()
	queueSize.WithLabelValues(m.queue).Dec()
}

func (m *Prometheus) QueueLeak() {
	queueLeak.WithLabelValues(m.queue).Inc()
}
