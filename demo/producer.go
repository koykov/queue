package main

type producerProc func(chan interface{}, interface{}, chan uint8)
