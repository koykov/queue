package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
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
		c   queue.Config
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

		err = json.Unmarshal(body, &c)
		if err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		c.MetricsHandler = prometheus.NewMetricsWriter(c.MetricsKey)

		var (
			procsMin, procsMax uint32
		)

		if pmin := r.FormValue("pmin"); len(pmin) > 0 {
			upmin, err := strconv.ParseUint(pmin, 10, 32)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			procsMin = uint32(upmin)
		}

		if pmax := r.FormValue("pmax"); len(pmax) > 0 {
			upmax, err := strconv.ParseUint(pmax, 10, 32)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			procsMax = uint32(upmax)
		}

		c.Proc = queue.DummyProc
		c.LeakyHandler = &queue.DummyLeak{}
		qi := queue.New(c)

		q := demoQueue{
			queue:        qi,
			producersMin: procsMin,
			producersMax: procsMax,
			producers:    make([]producer, procsMax),
			ctl:          make([]chan signal, procsMax),
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
