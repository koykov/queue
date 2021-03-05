package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/koykov/blqueue"
	"github.com/koykov/blqueue/metrics/prometheus"
)

type QueueHTTP struct {
	mux  sync.RWMutex
	pool map[string]*demoQueue
}

func NewQueueHTTP() *QueueHTTP {
	h := &QueueHTTP{
		pool: make(map[string]*demoQueue),
	}
	return h
}

func (h *QueueHTTP) get(key string) *demoQueue {
	h.mux.RLock()
	defer h.mux.RUnlock()
	if q, ok := h.pool[key]; ok {
		return q
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
	case r.URL.Path == "/api/v1/ping":
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))

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
			conf blqueue.Config
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
		conf.Proc = blqueue.DummyProc
		if req.AllowLeak {
			conf.LeakyHandler = &blqueue.DummyLeak{}
		}
		qi := blqueue.New(conf)

		q := demoQueue{
			key:          key,
			queue:        qi,
			producersMin: req.ProducersMin,
			producersMax: req.ProducersMax,
		}

		h.mux.Lock()
		h.pool[key] = &q
		h.mux.Unlock()

		q.Run()

		w.WriteHeader(http.StatusOK)
		if _, err = w.Write([]byte("ok")); err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

	case r.URL.Path == "/api/v1/producer-up" && q != nil:
		var delta uint32
		if d := r.FormValue("delta"); len(d) > 0 {
			ud, err := strconv.ParseUint(d, 10, 32)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			delta = uint32(ud)
		}
		if err := q.ProducerUp(delta); err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err = w.Write([]byte("ok")); err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

	case r.URL.Path == "/api/v1/producer-down" && q != nil:
		var delta uint32
		if d := r.FormValue("delta"); len(d) > 0 {
			ud, err := strconv.ParseUint(d, 10, 32)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			delta = uint32(ud)
		}
		if err := q.ProducerDown(delta); err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err = w.Write([]byte("ok")); err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

	case r.URL.Path == "/api/v1/stop":
		if q != nil {
			q.Stop()
		}

		// h.mux.Lock()
		// delete(h.pool, key)
		// h.mux.Unlock()

		w.WriteHeader(http.StatusOK)
		if _, err = w.Write([]byte("ok")); err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}
