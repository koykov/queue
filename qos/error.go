package qos

import "errors"

var (
	ErrNoQoS              = errors.New("no QoS config provided")
	ErrQoSUnknownAlgo     = errors.New("unknown QoS scheduling algorithm")
	ErrQoSNoEvaluator     = errors.New("no QoS priority evaluator provided")
	ErrQoSNoQueues        = errors.New("no QoS queues")
	ErrQoSSenseless       = errors.New("QoS is senseless")
	ErrQoSIngressReserved = errors.New("name 'ingress' is reserved")
	ErrQoSEgressReserved  = errors.New("name 'egress' is reserved")
)
