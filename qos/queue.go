package qos

// Queue represent QoS sub-queue config.
type Queue struct {
	// Name of sub-queue. Uses for metrics.
	Name string
	// Capacity of the sub-queue.
	// Mandatory param.
	Capacity uint64
	// Sub-queue weight.
	// Big value means that sub-queue will income more items.
	Weight uint64
}
