# Queue

A `queue` is a wrapper over Go channels that has the following features:
* balanced
* leaky
* retryable
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
