package cb

import "time"

type CircuitBreakerParameters struct {
	MaxRequests         uint32
	ConsecutiveFailures uint32
	Interval            time.Duration
	Timeout             time.Duration
}
