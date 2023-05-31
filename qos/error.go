package qos

import "errors"

var (
	ErrNoConfig     = errors.New("no QoS config provided")
	ErrUnknownAlgo  = errors.New("unknown QoS scheduling algorithm")
	ErrNoEvaluator  = errors.New("no QoS priority evaluator provided")
	ErrNoQueues     = errors.New("no QoS queues")
	ErrSenseless    = errors.New("QoS is senseless")
	ErrNameReserved = errors.New("names 'ingress' and 'egress' are reserved")
)
