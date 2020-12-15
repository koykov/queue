package metrics

import "github.com/prometheus/client_golang/prometheus"

type Prometheus struct {
	queue string

	workerIdle, workerActive, workerSleep *prometheus.CounterVec
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

	prometheus.MustRegister(m.workerIdle, m.workerActive, m.workerSleep)

	return m
}

func (m *Prometheus) WorkerSleep(_ uint32) {
	m.workerSleep.WithLabelValues(m.queue).Add(1)
	m.workerActive.WithLabelValues(m.queue).Add(-1)
}

func (m *Prometheus) WorkerWakeup(_ uint32) {
	m.workerActive.WithLabelValues(m.queue).Add(1)
	m.workerSleep.WithLabelValues(m.queue).Add(-1)
}

func (m *Prometheus) WorkerStop(_ uint32) {
	m.workerIdle.WithLabelValues(m.queue).Add(1)
	m.workerActive.WithLabelValues(m.queue).Add(-1)
}

func (m *Prometheus) QueuePut() {
	//
}

func (m *Prometheus) QueuePull() {
	//
}

func (m *Prometheus) QueueLeak() {
	//
}
