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

	allow400 map[string]bool
	allow404 map[string]bool
}

type QueueResponse struct {
	Status  int    `json:"status"`
	Error   string `json:"error"`
	Message string `json:"reponse"`
}

func NewQueueHTTP() *QueueHTTP {
	h := &QueueHTTP{
		pool: make(map[string]*demoQueue),
		allow400: map[string]bool{
			"/api/v1/ping": true,
		},
		allow404: map[string]bool{
			"/api/v1/init": true,
			"/api/v1/ping": true,
		},
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
		key  string
		q    *demoQueue
		resp QueueResponse
	)

	defer func() {
		w.WriteHeader(resp.Status)
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)
	}()

	resp.Status = http.StatusOK

	if key = r.FormValue("key"); len(key) == 0 && !h.allow400[r.URL.Path] {
		resp.Status = http.StatusBadRequest
		return
	}
	if q = h.get(key); q == nil && !h.allow404[r.URL.Path] {
		resp.Status = http.StatusNotFound
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	switch {
	case r.URL.Path == "/api/v1/ping":
		resp.Message = "pong"

	case r.URL.Path == "/api/v1/status" && q != nil:
		resp.Message = q.String()

	case r.URL.Path == "/api/v1/init":
		if q != nil {
			resp.Status = http.StatusNotAcceptable
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("err", err)
			resp.Status = http.StatusBadRequest
			resp.Error = err.Error()
			return
		}

		var (
			req  RequestInit
			conf blqueue.Config
		)

		err = json.Unmarshal(body, &req)
		if err != nil {
			log.Println("err", err)
			resp.Status = http.StatusBadRequest
			resp.Error = err.Error()
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

		resp.Message = "success"

	case r.URL.Path == "/api/v1/producer-up" && q != nil:
		var delta uint32
		if d := r.FormValue("delta"); len(d) > 0 {
			ud, err := strconv.ParseUint(d, 10, 32)
			if err != nil {
				log.Println("err", err)
				resp.Status = http.StatusInternalServerError
				resp.Error = err.Error()
				return
			}
			delta = uint32(ud)
		}
		if err := q.ProducerUp(delta); err != nil {
			log.Println("err", err)
			resp.Status = http.StatusInternalServerError
			resp.Error = err.Error()
			return
		}
		resp.Message = "success"

	case r.URL.Path == "/api/v1/producer-down" && q != nil:
		var delta uint32
		if d := r.FormValue("delta"); len(d) > 0 {
			ud, err := strconv.ParseUint(d, 10, 32)
			if err != nil {
				log.Println("err", err)
				resp.Status = http.StatusInternalServerError
				resp.Error = err.Error()
				return
			}
			delta = uint32(ud)
		}
		if err := q.ProducerDown(delta); err != nil {
			log.Println("err", err)
			resp.Status = http.StatusInternalServerError
			resp.Error = err.Error()
			return
		}
		resp.Message = "success"

	case r.URL.Path == "/api/v1/stop":
		if q != nil {
			q.Stop()
		}

		// h.mux.Lock()
		// delete(h.pool, key)
		// h.mux.Unlock()

		resp.Message = "success"
	default:
		resp.Status = http.StatusNotFound
		return
	}
}
