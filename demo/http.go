package main

import (
	"log"
	"net/http"
	"strconv"
	"time"
)

type QueueHTTP struct{}

func (h *QueueHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error

	switch {
	case r.URL.Path == "/api/v1/status":
		w.WriteHeader(http.StatusOK)
		if _, err = w.Write([]byte("ok")); err != nil {
			log.Println("err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case r.URL.Path == "/api/v1/init":
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
			WakeupFactor = float32(fwakeup)
		}

		if sleep := r.FormValue("sleep"); len(sleep) > 0 {
			fsleep, err := strconv.ParseFloat(sleep, 32)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			SleepFactor = float32(fsleep)
		}

		if hb := r.FormValue("hb"); len(hb) > 0 {
			ihb, err := strconv.ParseInt(hb, 10, 64)
			if err != nil {
				log.Println("err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			Heartbeat = time.Duration(ihb)
		}

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
