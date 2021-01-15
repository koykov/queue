package main

import "github.com/prometheus/client_golang/prometheus"

var (
	producerIdle, producerActive *prometheus.GaugeVec
)

func init() {
	producerIdle = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_producers_idle",
		Help: "Indicates how many producers idle.",
	}, []string{"queue"})
	producerActive = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_producers_active",
		Help: "Indicates how many producers active.",
	}, []string{"queue"})

	prometheus.MustRegister(producerIdle, producerActive)
}

func ProducerStartMetric(queue string) {
	producerActive.WithLabelValues(queue).Inc()
	producerIdle.WithLabelValues(queue).Add(-1)
}

func ProducerStopMetric(queue string) {
	producerIdle.WithLabelValues(queue).Inc()
	producerActive.WithLabelValues(queue).Add(-1)
}
