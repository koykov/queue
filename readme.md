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
* _sleep_ - worker is active, but does nothing due to queue hasn't enouht items to process. This state isn't permanent -
after waiting `SleepInterval` worker become idle.
* _idle_ - worker (goroutine) stops and release resources. By need queue may make idle worker to active (run goroutine).

Queue makes a decision to run new worker when [rate](https://github.com/koykov/queue/blob/master/interface.go#L12)
became greather than `WakeupFactor` [0..0.999999].
