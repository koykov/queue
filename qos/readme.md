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

### Output (egress) SQ

`egress` is a special SQ, where puts items taken from other SQs. There is param `EgressConfig` to set up it:
* `Capacity` - capacity, mandatory.
* `Streams` - shards of egress.
* `Workers` - count of workers that will take items from SQs and put to egress.
* `IdleThreshold` - how many idle reads available from empty SQs before worker blocks for `IdleTimeout`.
* `IdleTimeout` - how long worker will block after `IdleThreshold` failed read attempts.

Each egress worker works iteratively and reads one item per turn. From what SQ item will read dependents of prioritization
algorithm (see param `Algo`). If there is no items to read, the iteration marked as "blank" (idle). After `IdleThreshold`
blank reads worker will block for `IdleTimeout` period. But worker may unblock earlier, if item will enqueue.

### Priority evaluator (PE)

Param `Evaluator` implements `PriorityEvaluator` interface and helps to "evaluate" priority of item in percent. In
dependency of priority item will put to one of SQs (according weight).

There are to builtin evaluators:
* [`Weighed`](https://github.com/koykov/queue/blob/master/priority/weighted.go) - for items given or calculated weight.
* [`Random`](https://github.com/koykov/queue/blob/master/priority/random.go) - testing PE with random values.
