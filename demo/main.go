package main

import (
	"log"
	"net/http"
	"time"
)

var (
	workersMin, workersMax    uint32
	WakeupFactor, SleepFactor float32
	Heartbeat                 time.Duration

	qhandler QueueHTTP
)

func main() {
	srv := &http.Server{
		Addr:    ":8080",
		Handler: &qhandler,
	}
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal("err", err)
	}
}
