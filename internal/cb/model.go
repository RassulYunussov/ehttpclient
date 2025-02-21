package cb

import (
	"time"
)

const errMessage = "5xx http status error"

type CircuitBreakerParameters struct {
	MaxRequests         uint32
	ConsecutiveFailures uint32
	Interval            time.Duration
	Timeout             time.Duration
}

type circuitBreakerErrorWrapper[T any] struct {
	wrapped T
}

func (c *circuitBreakerErrorWrapper[T]) Error() string {
	return errMessage
}
