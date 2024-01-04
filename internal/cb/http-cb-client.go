package cb

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/RassulYunussov/ehttpclient/internal/resilient"
	"github.com/sony/gobreaker"
)

type CircuitBreakerHttpClient interface {
	DoResourceRequest(resource string, r *http.Request) (*http.Response, error)
	Do(r *http.Request) (*http.Response, error)
}

type circuitBreakerBackedHttpClient struct {
	sync.Mutex
	resilientHttpClient resilient.ResilientHttpClient
	maxRequests         uint32
	consecutiveFailures uint32
	interval            time.Duration
	timeout             time.Duration
	circuitBreakers     map[string]*gobreaker.CircuitBreaker
}

func CreateCircuitBreakerHttpClient(resilientHttpClient resilient.ResilientHttpClient, circuitBreakerParameters *CircuitBreakerParameters) CircuitBreakerHttpClient {
	return &circuitBreakerBackedHttpClient{
		maxRequests:         circuitBreakerParameters.MaxRequests,
		consecutiveFailures: circuitBreakerParameters.ConsecutiveFailures,
		interval:            circuitBreakerParameters.Interval,
		timeout:             circuitBreakerParameters.Timeout,
		Mutex:               sync.Mutex{},
		resilientHttpClient: resilientHttpClient,
		circuitBreakers:     make(map[string]*gobreaker.CircuitBreaker),
	}
}

func (c *circuitBreakerBackedHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	cb := c.getCircuitBreaker(resource)
	resp, err := cb.Execute(func() (interface{}, error) {
		return c.resilientHttpClient.Do(r)
	})
	if err != nil {
		return nil, err
	}
	return resp.(*http.Response), nil
}

func (c *circuitBreakerBackedHttpClient) Do(r *http.Request) (*http.Response, error) {
	return c.DoResourceRequest(getResource(r), r)
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
			return counts.ConsecutiveFailures >= c.consecutiveFailures
		},
	})
	c.circuitBreakers[resource] = cb
	return cb
}

func getResource(r *http.Request) string {
	return r.Method + "_" + r.URL.Path
}
