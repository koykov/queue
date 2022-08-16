# Balanced Leaky Queue

`blqueue` balances queue among available workers (`WorkersMin` and `WorkersMax` params).

It has the following features:
* Put to sleep idle workers depending of queue's fullness rate.
* Resume sleeping (or start idle) workers when fullness rate of the queue grows over the limit.
* Use [leaky bucket](https://en.wikipedia.org/wiki/Leaky_bucket) algorithm on newly comes elements when queue is full.
* Leaked elements puts to DLQ (dead letter queue) to perform some actions on them.
* Retry items processing.
* Schedule active workers count.
* Delayed execution.

## See

* https://en.wikipedia.org/wiki/Leaky_bucket
* https://golang.org/doc/effective_go#leaky_buffer
* https://en.wikipedia.org/wiki/Thread_pool
