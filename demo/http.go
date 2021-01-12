package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/koykov/queue"
	"github.com/koykov/queue/metrics/prometheus"
)

type QueueHTTP struct {
	mux  sync.RWMutex
	pool map[string]demoQueue
}

func NewQueueHTTP() *QueueHTTP {
	h := &QueueHTTP{
		pool: make(map[string]demoQueue),
	}
	return h
}

func (h *QueueHTTP) get(key string) *demoQueue {
	h.mux.RLock()
	defer h.mux.RUnlock()
	if q, ok := h.pool[key]; ok {
		return &q
	}
	return nil
}

func (h *QueueHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		key string
		err error
		q   *demoQueue
	)

	if key = r.FormValue("key"); len(key) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if q = h.get(key); q == nil && r.URL.Path != "/api/v1/init" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.URL.Path == "/api/v1/status" && q != nil:
		w.WriteHeader(http.StatusOK)
		if _, err = w.Write([]byte(q.String())); err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case r.URL.Path == "/api/v1/init":
		if q != nil {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var (
			req  RequestInit
			conf queue.Config
		)

		err = json.Unmarshal(body, &req)
		if err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(req.MetricsKey) == 0 {
			req.MetricsKey = key
		}

		conf.Size = req.Size
		conf.Workers = req.Workers
		conf.Heartbeat = req.Heartbeat
		conf.WorkersMin = req.WorkersMin
		conf.WorkersMax = req.WorkersMax
		conf.WakeupFactor = req.WakeupFactor
		conf.SleepFactor = req.SleepFactor
		conf.MetricsKey = req.MetricsKey

		conf.MetricsHandler = prometheus.NewMetricsWriter(conf.MetricsKey)
		conf.Proc = queue.DummyProc
		if req.AllowLeak {
			conf.LeakyHandler = &queue.DummyLeak{}
		}
		qi := queue.New(conf)

		q := demoQueue{
			queue:        qi,
			producersMin: req.ProducersMin,
			producersMax: req.ProducersMax,
			producers:    make([]producer, req.ProducersMax),
			ctl:          make([]chan signal, req.ProducersMax),
		}

		h.mux.Lock()
		h.pool[key] = q
		h.mux.Unlock()

		q.Run()

		w.WriteHeader(http.StatusOK)
		if _, err = w.Write([]byte("ok")); err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}
