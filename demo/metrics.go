package main

import "github.com/prometheus/client_golang/prometheus"

var (
	producerIdle, producerActive, producerSleep *prometheus.GaugeVec
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
	producerSleep = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_producers_sleep",
		Help: "Indicates how many producers sleep.",
	}, []string{"queue"})

	prometheus.MustRegister(producerIdle, producerActive, producerSleep)
}

func ProducerSleep(queue string) {
	producerSleep.WithLabelValues(queue).Inc()
	producerActive.WithLabelValues(queue).Add(-1)
}

func ProducerWakeup(queue string) {
	producerActive.WithLabelValues(queue).Inc()
	producerSleep.WithLabelValues(queue).Add(-1)
}

func ProducerStop(queue string) {
	producerIdle.WithLabelValues(queue).Inc()
	producerActive.WithLabelValues(queue).Add(-1)
}
