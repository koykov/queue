package metrics

import "github.com/prometheus/client_golang/prometheus"

type Prometheus struct {
	queue string

	queueSize *prometheus.GaugeVec

	workerIdle, workerActive, workerSleep,
	queueIn, queueOut, queueLeak *prometheus.CounterVec
}

func NewPrometheusMetrics(queueKey string, workersMax uint32) *Prometheus {
	m := &Prometheus{}
	m.workerIdle = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_workers_idle",
		Help: "Indicates how many workers idle.",
	}, []string{"queue"})
	m.workerActive = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_workers_active",
		Help: "Indicates how many workers active.",
	}, []string{"queue"})
	m.workerSleep = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_workers_sleep",
		Help: "Indicates how many workers sleep.",
	}, []string{"queue"})

	m.queueSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_size",
		Help: "Actual queue size.",
	}, []string{"queue"})

	m.queueIn = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_in",
		Help: "How many items comes to the queue.",
	}, []string{"queue"})
	m.queueOut = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_out",
		Help: "How many items leaves queue.",
	}, []string{"queue"})
	m.queueLeak = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_leak",
		Help: "How many items dropped on the floor due to queue is full.",
	}, []string{"queue"})

	prometheus.MustRegister(m.workerIdle, m.workerActive, m.workerSleep)

	return m
}

func (m *Prometheus) WorkerSleep(_ uint32) {
	m.workerSleep.WithLabelValues(m.queue).Inc()
	m.workerActive.WithLabelValues(m.queue).Add(-1)
}

func (m *Prometheus) WorkerWakeup(_ uint32) {
	m.workerActive.WithLabelValues(m.queue).Inc()
	m.workerSleep.WithLabelValues(m.queue).Add(-1)
}

func (m *Prometheus) WorkerStop(_ uint32) {
	m.workerIdle.WithLabelValues(m.queue).Inc()
	m.workerActive.WithLabelValues(m.queue).Add(-1)
}

func (m *Prometheus) QueuePut() {
	m.queueIn.WithLabelValues(m.queue).Inc()
	m.queueSize.WithLabelValues(m.queue).Inc()
}

func (m *Prometheus) QueuePull() {
	m.queueOut.WithLabelValues(m.queue).Inc()
	m.queueSize.WithLabelValues(m.queue).Dec()
}

func (m *Prometheus) QueueLeak() {
	m.queueLeak.WithLabelValues(m.queue).Inc()
}
