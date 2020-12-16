package main

import (
	"log"
	"net/http"
)

var (
	qh *QueueHTTP
)

func init() {
	qh = NewQueueHTTP()
}

func main() {
	if err := http.ListenAndServe(":8080", qh); err != nil {
		log.Fatal(err)
	}
}
