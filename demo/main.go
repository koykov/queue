package main

import (
	"flag"
	"log"
	"time"
)

var (
	fWmin   = flag.Uint("wmin", 0, "Minimal workers count")
	fWmax   = flag.Uint("wmax", 0, "Maximum workers count")
	fWakeup = flag.Float64("wakeup", .7, "Wakeup worker factor")
	fSleep  = flag.Float64("sleep", .5, "Sleep worker factor")
	fHb     = flag.Int64("hb", 0, "Heartbeat delay")

	workersMin, workersMax    uint32
	wakeupFactor, sleepFactor float32
	Heartbeat                 time.Duration
)

func init() {
	flag.Parse()

	workersMin = uint32(*fWmin)
	workersMax = uint32(*fWmax)
	wakeupFactor = float32(*fWakeup)
	sleepFactor = float32(*fSleep)
	Heartbeat = time.Duration(*fHb)
}

func main() {
	log.Printf("queue demo\noptions:\n * workers min limit - %d\n * workers max limit - %d\n * wakeup factor - %f\n * sleep factor - %f\n * heartbeat - %d",
		workersMin, workersMax, wakeupFactor, sleepFactor, Heartbeat)
}
