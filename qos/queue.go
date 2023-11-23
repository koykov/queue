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
	// Mandatory param if IngressWeight/EgressWeight omitted.
	Weight uint64
	// Ingress weight of sub-queue.
	// Mandatory param if Weight omitted.
	IngressWeight uint64
	// Egress weight of sub-queue.
	// Mandatory param if Weight omitted.
	EgressWeight uint64
}
