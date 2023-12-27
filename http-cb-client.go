package ehttpclient

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/sony/gobreaker"
)

type circuitBreakerBackedHttpClient struct {
	sync.Mutex
	*resilientHttpClient
	MaxRequests         uint32
	ConsecutiveFailures uint32
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
		MaxRequests: c.MaxRequests,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > c.ConsecutiveFailures
		},
	})
	c.circuitBreakers[resource] = cb
	return cb
}

func (c *circuitBreakerBackedHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	cb := c.getCircuitBreaker(resource)
	resp, err := cb.Execute(func() (interface{}, error) {
		return c.doWithRetry(resource, r)
	})
	if err != nil {
		return nil, err
	}
	return resp.(*http.Response), nil
}

func (c *circuitBreakerBackedHttpClient) Do(r *http.Request) (*http.Response, error) {
	return c.DoResourceRequest(getResource(r), r)
}
