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
