# TODO

* ~~Implement Queue interface~~
```go
type Interface interface {
	Enqueue(x interface{}) error
	EnqueueWithDelay(x interface{}, delay time.Duration) error // or ...WithDE (delayed execution)
	Rate() float32
	Close()
	ForceClose()
}
```
* ~~Implement Worker interface~~
```go
type Worker interface {
	Do(x interface{}) error
	DoContext(ctx context.Context, x interface{}) error // idea
}
```
* ~~Implement WorkerComposer that implements Worker interface and run all registered workers sequentially~~
```go
wc := NewWorkerComposer().Add(&worker0).
	Add(&worker1).
	...
        Add(&workerN)
// or
wc := NewWorkerComposer(&worker0, &worker1, ...)
// or
wc := ComposeWorkers(&worker0, &worker1, ...)

conf.Worker = &wc
```
* ~~Implement TransitWorker to route item to other queue~~
```go
type TransitWorker struct {
	Queue QueueInterface
	Do(x interface{}) error {
	    _ = tw.Queue.Enqueue(x)	
    }
}
```
* Add Config.Timeout to limit processing time.
  * Consider WorkerComposer.
  * Think about dumps.
* ~~Change Config.DLQ type to QueueInterface~~
* ~~Implement TrashDLQ to put items to /dev/null~~
* Implement DumpDLQ with following features:
  * Allow to specify max size in bytes
  * Allow to specify max wait time (since first income)
  * Allow to specify queue to enqueue restored items
  * Allow to take items that implement
    * MarshallerTo (MarshalTo() + Size()) - protobuf case
    * Marshaller
    * Stringer
  * Write to binary file (version + header + payload)
  * Scheduled worker that must
    * Read dump file (unmarshal each item)
    * Put to queue if possible (rate less than some limit)
  * Idea: specify EncoderDecoder (MarshallerUnmarshaller) interface that will do endec logic.
* Rename package, variants:
  * queue
  * qtk (queue toolkit)
  * pipe/pipeq
  * ...
* Comment issues.
* Add detailed readme.
