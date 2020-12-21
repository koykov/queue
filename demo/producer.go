package main

type status uint
type signal uint

const (
	statusIdle   status = 0
	statusActive        = 1
)

type producerProc func(chan interface{}, interface{}, chan signal)

type producer struct {
	status status
	proc   producerProc
}
