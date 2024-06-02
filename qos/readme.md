# QoS

QoS (Quality of Service) allows to prioritize items in the queue and process them in the order according prioritization
algorithm.

QoS uses a sub-queues (SQ) for prioritization, each of which is a primitive FIFO queue. Items from SQ, according
prioritization algorithm, forwards to special egress SQ and then forwards to workers.

To enable the feature you must set up param `QoS` in queue's config, like this:
```go
conf := Config{
	...
	QoS: &qos.Config{ ... } // или использовать qos.New(...)
	...
}
```

> Note, setting up this params will overwrite `Capacity` params with total sum of all SQs.

## Settings

### Prioritization algorithm

Param `Algo` in QoS config defines from what SQ the next item will take to forward to egress. Currently, supports three
algorithms:
* `PQ` (Priority Queuing) - the SQ that is specified first will process first, the second SQ after first become empty, ...
* `RR` (Round-Robin) - items will take from every SQs in rotation every turn.
* `WRR` (Weighted Round-Robin) - items forwards to egress from SQ according it weight.

`DWRR` (Dynamic Weighted Round-Robin) isn't implemented but planned.
