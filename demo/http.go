package main

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

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

		var (
			size               uint64
			procsMin, procsMax uint32

			workers, workersMin, workersMax uint32

			wakeupFactor, sleepFactor float32

			heartbeat time.Duration
			metrics   = prometheus.NewMetricsWriter(key)
		)

		if qsize := r.FormValue("size"); len(qsize) > 0 {
			uqsize, err := strconv.ParseUint(qsize, 10, 32)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			size = uqsize
		}

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

		if work := r.FormValue("work"); len(work) > 0 {
			uwork, err := strconv.ParseUint(work, 10, 32)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			workers = uint32(uwork)
		}

		if wmin := r.FormValue("wmin"); len(wmin) > 0 {
			uwmin, err := strconv.ParseUint(wmin, 10, 32)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			workersMin = uint32(uwmin)
		}

		if wmax := r.FormValue("wmax"); len(wmax) > 0 {
			uwmax, err := strconv.ParseUint(wmax, 10, 32)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			workersMax = uint32(uwmax)
		}

		if wakeup := r.FormValue("wakeup"); len(wakeup) > 0 {
			fwakeup, err := strconv.ParseFloat(wakeup, 32)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			wakeupFactor = float32(fwakeup)
		}

		if sleep := r.FormValue("sleep"); len(sleep) > 0 {
			fsleep, err := strconv.ParseFloat(sleep, 32)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			sleepFactor = float32(fsleep)
		}

		if hb := r.FormValue("hb"); len(hb) > 0 {
			ihb, err := strconv.ParseInt(hb, 10, 64)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			heartbeat = time.Duration(ihb)
		}

		var qi queue.Queuer
		typ := r.FormValue("type")
		switch typ {
		case "bqueue":
			qi = &queue.BalancedQueue{
				Queue: queue.Queue{
					Size:    size,
					Key:     key,
					Metrics: metrics,
				},
				WorkersMin:   workersMin,
				WorkersMax:   workersMax,
				WakeupFactor: wakeupFactor,
				SleepFactor:  sleepFactor,
				Heartbeat:    heartbeat,
			}
		case "blqueue":
			qi = &queue.BalancedLeakyQueue{
				BalancedQueue: queue.BalancedQueue{
					Queue: queue.Queue{
						Size:    size,
						Key:     key,
						Metrics: metrics,
					},
					WorkersMin:   workersMin,
					WorkersMax:   workersMax,
					WakeupFactor: wakeupFactor,
					SleepFactor:  sleepFactor,
					Heartbeat:    heartbeat,
				},
				Leaker: nil,
			}
		case "queue":
			fallthrough
		default:
			qi = &queue.Queue{
				Size:    size,
				Key:     key,
				Workers: workers,
				Metrics: metrics,
			}
		}

		q := demoQueue{
			queue:        qi,
			producersMin: procsMin,
			producersMax: procsMax,
			producers:    make([]producer, procsMax),
			ctl:          make([]chan uint8, procsMax),
		}

		h.mux.Lock()
		h.pool[key] = q
		h.mux.Unlock()

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
