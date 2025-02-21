package cb

import (
	"fmt"
	"time"

	"github.com/sony/gobreaker/v2"
)

type circuitBreaker[T any, V any] struct {
	*gobreaker.CircuitBreaker[*V]
}

func (cb *circuitBreaker[T, V]) execute(f func(request *T) (*V, error), request *T) (*V, error) {
	res, err := cb.CircuitBreaker.Execute(func() (*V, error) {
		return f(request)
	})
	if err != nil {
		return nil, err
	}
	return res, err
}

func newCircuitBreaker[T any, V any](maxRequests uint32, interval time.Duration, timeout time.Duration, consecutiveFailures uint32, resource string) *circuitBreaker[T, V] {
	return &circuitBreaker[T, V]{
		CircuitBreaker: gobreaker.NewCircuitBreaker[*V](gobreaker.Settings{
			Name:        fmt.Sprintf("http client circuit breaker for resource %s", resource),
			MaxRequests: maxRequests,
			Interval:    interval,
			Timeout:     timeout,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= consecutiveFailures
			},
		}),
	}
}
