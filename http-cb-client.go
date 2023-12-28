package ehttpclient

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

type circuitBreakerBackedHttpClient struct {
	sync.Mutex
	*resilientHttpClient
	maxRequests         uint32
	consecutiveFailures uint32
	interval            time.Duration
	timeout             time.Duration
	circuitBreakers     map[string]*gobreaker.CircuitBreaker
}

func (c *circuitBreakerBackedHttpClient) getCircuitBreaker(resource string) *gobreaker.CircuitBreaker {
	c.Lock()
	defer c.Unlock()
	if cb, ok := c.circuitBreakers[resource]; ok {
		return cb
	}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        fmt.Sprintf("http client circuit breaker for resource %s", resource),
		MaxRequests: c.maxRequests,
		Interval:    c.interval,
		Timeout:     c.timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > c.consecutiveFailures
		},
	})
	c.circuitBreakers[resource] = cb
	return cb
}

func (c *circuitBreakerBackedHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	cb := c.getCircuitBreaker(resource)
	resp, err := cb.Execute(func() (interface{}, error) {
		return c.doWithRetry(r)
	})
	if err != nil {
		return nil, err
	}
	return resp.(*http.Response), nil
}

func (c *circuitBreakerBackedHttpClient) Do(r *http.Request) (*http.Response, error) {
	return c.DoResourceRequest(getResource(r), r)
}
