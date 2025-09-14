# Queue

A `queue` is a wrapper over Go channels that has the following features:
* balanced
* leaky
* retryable
* backoff support
* jitter support
* scheduled
* delayed execution
* deadline-aware
* prioretizable
* covered with metrics
* logged

`queue` was developed in response to a need to create a lot of queues with the same structure and functionality. Create
identical channels with only different workers, cover them with metrics was too boring and as result this solution was born.

`queue` allows to abstract from the implementation of the channel and producers and focus only on worker implementation.
It's enough to write a worker that implements [Worker](https://github.com/koykov/queue/blob/master/interface.go#L18),
bind it to the [config](https://github.com/koykov/queue/blob/master/config.go#L22) of the queue, and it will do all work
itself.

`queue` isn't a classic queue with `Enqueue`/`Dequeue` methods. This implementation hasn't `Dequeue` method due to queue
extracts items itself and forward them to any of active workers. That implementation is similar to
[`thread pool`](https://en.wikipedia.org/wiki/Thread_pool) template, but `queue` go beyond frames of this template.

Queue initializes using [`Config`](https://github.com/koykov/queue/blob/master/config.go#L22). It has only two necessary
settings - `Capacity` and `Worker`. `Worker` must implement [Worker](https://github.com/koykov/queue/blob/master/interface.go#L18)
interface. `Worker` can only process the item and return error if occurred. Workers count can be setup using `Workers`
param. If you want dynamic workers count, i.e. balanced queue, count should be setup using `WorkersMin`/`WorkersMax`
params (see balanced queue chapter).

As result queue will work as classic thread pool with static workers count.

Let's see how enables and works other features of the `queue`.

## Balanced queue

Usually all queues has variadic load - maximum at the day, minimum at the night. The maximum number of workers must be
active independent of current load. It isn't a big problem due to goroutines is cheap, but solving the problem of 
balancing count of workers depending on load was too interesting and therefore this feature was implemented.

Balancing enables by setting up params `WorkersMin` and `WorkersMax` (`WorkersMin` must be less that `WorkersMax`).
Setting up this params disables `Workers` param, i.e. balancing feature has high priority than static workers count.

`WorkersMin` limits low count of active workers. Independent of conditions the queue will keep that number of workers
active.

`WorkersMax` limits maximum count of active workers. Queue wouldn't run more workers than `WorkersMax`. Even if
`WorkersMax` workers isn't enough, then leaky feature may help (see chapter Leaky queue).

All workers in range `WorkersMin` - `WorkersMax` may have three state:
* _active_ - worker works and processes the items.
* _sleep_ - worker is active, but does nothing due to queue hasn't enough items to process. This state isn't permanent -
after waiting `SleepInterval` worker become idle.
* _idle_ - worker (goroutine) stops and release resources. By need queue may make idle worker to active (run goroutine).

Queue makes a decision to run new worker when [rate](https://github.com/koykov/queue/blob/master/interface.go#L12)
became greather than `WakeupFactor` [0..0.999999].

Eg: let's imagine queue with capacity 100 and `WakeupFactor` 0.5. If queue size will greater than 50, the queue will run
new worker. If new worker wouldn't help to reduce size, queue will start another one till rate become less than
`WakeupFactor` or `WorkersMax` limit reached.

Let's imagine next that queue's load reduces and count of active workers became redundant. In that case queue will check
`SleepFactor` [0..0.999999]. If queue's rate become less that `SleepFactor` one of active workers will force to sleep
state. Next check another on will sleep, if condition (rate < `SleepFactor`) keep true - till rate will greater that
`SleepFactor` or `WorkersMin` limit reaches. Sleeping worker will not sleep forever. After waiting `SleepInterval`
his goroutine will stop and status become _idle_. Sleeping state is required for queues with often variadic load.
Permanent goroutine running/stopping triggers `runtime.findrunnable` function. `SleepInterval` helps amortize that load. 

Queue in balancing mode permanent balances workers count so that queue's rate is between `SleepFactor` and `WakeupFactor`.

## Leaky queue

Let's imagine a queue with so huge load, that even `WorkersMax` active can't process the items in time. The queue blocks
all threads calling `Enqueue`, that may produces deadlock or reduce performance.

For solving this problem was implemented `DLQ` (dead letter queue) - an auxiliary component, implements
[Enqueuer](https://github.com/koykov/queue/blob/master/interface.go#L4) interface. Thus, you may forward leaked items to
another queue or even make a chain of queues.

Setting up param `DLQ` in config enables "leaky" feature of the queue. It based on
["leaky bucket algorithm"](https://en.wikipedia.org/wiki/Leaky_bucket). It described in
[Effective Go](https://golang.org/doc/effective_go#leaky_buffer) as "leaky buffer".

Package contains builtin [Dummy DLQ](https://github.com/koykov/queue/blob/master/dummy.go#L23) implementation. It just
throws leaked items to the trash - you will lose some items, but will keep queue and application alive. However, there are
[dlqdump](https://github.com/koykov/dlqdump) solution, that may dump leaked items to some storage (eg: disk). See package
description for details.

Final note of leaky queue: there is config flag `FailToDLQ`. If worker reports that item processing fails, the item will
forward to `DLQ`, even if queue isn't leaked at the moment. It may be helpful for to make fallback method of item processing.

## Retryable

One attempt of item processing may be not enough. For example, queue must send HTTP request and sending in worker fails
due to network problem and makes sense to try again. Param `MaxRetries` indicates how many repeated attempts worker can
take. The first attempt of processing isn't a retry. All next attempts interpreted as retry.

This param may work together with `FailToDLQ` param. Item will send to DLQ if all repeated attempts fails.

## Backoff

Continuation of the previous chapter.

If workers start throwing errors and mass retry is required, the queue may not be able to handle the load of both
incoming elements and retryable elements. For this case, two parameters have been added:

* `RetryInterval` - delay between processing attempts
* `Backoff` - a special component that calculates the real delay time based on `RetryInterval`

The following backoffs are available out of the box:

* [Linear](backoff/linear.go) (1, 2, 3, 4, ...)
* [Exponential](backoff/exponential.go) (1, 2, 4, 8, ...)
* [Quadratic](backoff/quadratic.go) (1, 4, 9, 16, ...)
* [Polynomial](backoff/polynomial.go) (1, 8, 27, ...)
* [Logarithmic](backoff/logarithmic.go) (for a delay of 10: 7, 11, 14, ...)
* [Fibonacci](backoff/fibonacci.go) (1, 1, 2, 3, 5, 8, ...)
* [Random](backoff/random.go) (`RetryInterval/2 + random(RetryInterval)`)

## Jitter support

Continuation of the previous point.

In case of mass retries, they can be synchronized in time and will go in waves, which will prevent the queue from processing them.
Especially for this case, there is a `Jitter` parameter, which allows you to "smear" retries in time by adding a random delay to the retry time.

The parameter only works with a non-zero `RetryInterval` and can work together with `Backoff`.

The following jitters are available out of the box:

* [Full](jitter/full.go) - returns a random value in the interval [0..RetryInterval).
* [Half](jitter/half.go) - returns a random value in the interval [RetryInterval/2 .. RetryInterval].
* [Decorrelated](jitter/decorrelated.go) - a random value between [Min..Max] taking into account the previous value.

## Scheduled queue

Let's imagine we know periodicity of growing/reducing of queue load. For example, from 8:00 AM till 12:00 AM and from
04:00 PM till 06:00 PM the load is moderate and 5 workers is enough. From 12:00 AM till 04:00 PM the load is maximal and
minimum  10 workers must be active. And at night the load is lowes and one worker is enough. For that cases was implemented
[scheduler](https://github.com/koykov/queue/blob/master/schedule.go) of queue params. It allows to set more optimal
values of the following params for certain periods of time:
* `WorkersMin`
* `WorkersMax`
* `WakeupFactor`
* `SleepFactor`

These params replaces corresponding config's value in the given period.

For above example, the scheduler initialization look the following:
```go
sched := NewSchedule()
sched.AddRange("08:00-12:00", ScheduleParams{WorkersMin: 5, WorkersMax: 10})
sched.AddRange("12:00-16:00", ScheduleParams{WorkersMin: 10, WorkersMax: 20})
sched.AddRange("16:00-18:00", ScheduleParams{WorkersMin: 5, WorkersMax: 10})
config := Config{
	...
	WorkersMin: 1,
	WorkersMax: 4,
	Schedule: sched,
	...
}
```
This config will balance queue for periods:
* from 5 to 10 active workers in period 8:00 AM - 12:00 AM
* from 10 to 20 active workers in period 12:00 AM - 04:00 PM
* from 5 to 10 active workers in period 04:00 PM - 06:00 PM
* from 1 to 4 active workers in the rest of time

The reason of this feature development is balances simplification in hot periods.  

## Delayed execution queue (DEQ)

If queue must process item not immediately after enqueue, but after a period you may use param `DelayInterval`. Setting
this param enables DEQ feature and guarantees that item will process after at least `DelayInterval` period.

This param is opposite to `DeadlineInterval`.

## Deadline-aware queue (DAQ)

In high-loaded queues the delivery of item to worker may take so much time that processing loss the meaning. The param
`DeadlineInterval` may help in that case. If item acquires by worker, but `DeadlineInterval` (since enqueue) already
passed, the item will not process.

This params may work together with `DeadlineToDLQ` flag.

This param is opposite to `DelayInterval`.

## Prioretizable queue

By default, queue works as FIFO stack. It works good while queue gets items with the same priority. But if queue receives
items of both types - priority and non-priority sooner or later will happen the case when queue will have non-priority
elements in the head and priority in the tail. Priority items may become outdated when they are turn and their processing
will lose maindness. In computer networks this problem was solved a long time ago and solution calls
[Quality of Service (QoS)](https://en.wikipedia.org/wiki/Quality_of_service).

The config has param `QoS` with type [`qos.Config`](qos/config.go). Setting up this param makes the queue prioretizable.
See configuration details in [readme](qos/readme.md).

## Metrics coverage

Config has a param calls `MetricsWriter` that must implement [`MetricsWriter`](https://github.com/koykov/queue/blob/master/metrics.go#L7)
interface.

There are two implementation of the interface:
* [`log.MetricsWriter`](metrics/log)
* [`prometheus.MetricsWriter`](metrics/prometheus)

The first is useless in production and may be need only for debugging purposes. Second one is totally tested and works
well. You may write your own implementation of `MetricsWriter` for any required TSDB.

## Builtin workers

`queue` has three helper workers:
* [transit](https://github.com/koykov/queue/blob/master/worker/transit.go) just forwards the item to another queue.
* [chain](https://github.com/koykov/queue/blob/master/worker/chain.go) joins several workers to one. The item will
synchronously processed by all "child" workers. You may, for example, build a chain of workers and finish it with
`transit` worker.
* [async_chain](https://github.com/koykov/queue/blob/master/worker/async_chain.go) also joins workers into one, but item will process asynchronously by "child" workers. 

## Logging

`queue` may report about internal events (calibration(balancing), closing, worker signals, ...) for debugging purposes.
There is param `Logger` in config that must implement [`Logger`](https://github.com/koykov/queue/blob/master/logger.go)
interface.

## Showcase

During development the biggest problem was a covering with tests. Due to impossibility of unit-testing the 
[demo showcase](https://github.com/koykov/demo/tree/master/queue) project was developed, where were tested different
scenarios of queue configs. The project has Docker-container, including `Grafana`, `Prometheus` and `queue daemon`.
The project controls using HTTP requests, see [readme](https://github.com/koykov/demo/blob/master/queue/readme.md). 

Typical sets of requests https://github.com/koykov/demo/tree/master/queue/request.

## Links

* https://en.wikipedia.org/wiki/Thread_pool
* https://en.wikipedia.org/wiki/Leaky_bucket
* https://golang.org/doc/effective_go#leaky_buffer
* https://en.wikipedia.org/wiki/Dead_letter_queue
* https://cloud.yandex.com/en/docs/message-queue/concepts/dlq
* https://en.wikipedia.org/wiki/Quality_of_service
