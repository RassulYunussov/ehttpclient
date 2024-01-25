package cb

import (
	"net/http"
	"sync"
	"time"

	"github.com/RassulYunussov/ehttpclient/internal/resilient"
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
	circuitBreakers     map[string]*circuitBreaker[http.Request, http.Response]
}

func CreateCircuitBreakerHttpClient(resilientHttpClient resilient.ResilientHttpClient, circuitBreakerParameters *CircuitBreakerParameters) CircuitBreakerHttpClient {
	return &circuitBreakerBackedHttpClient{
		maxRequests:         circuitBreakerParameters.MaxRequests,
		consecutiveFailures: circuitBreakerParameters.ConsecutiveFailures,
		interval:            circuitBreakerParameters.Interval,
		timeout:             circuitBreakerParameters.Timeout,
		Mutex:               sync.Mutex{},
		resilientHttpClient: resilientHttpClient,
		circuitBreakers:     make(map[string]*circuitBreaker[http.Request, http.Response]),
	}
}

func (c *circuitBreakerBackedHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	cb := c.getCircuitBreaker(resource)
	return cb.execute(c.resilientHttpClient.Do, r)
}

func (c *circuitBreakerBackedHttpClient) Do(r *http.Request) (*http.Response, error) {
	return c.DoResourceRequest(getResource(r), r)
}

func (c *circuitBreakerBackedHttpClient) getCircuitBreaker(resource string) *circuitBreaker[http.Request, http.Response] {
	c.Lock()
	defer c.Unlock()
	if cb, ok := c.circuitBreakers[resource]; ok {
		return cb
	}
	cb := newCircuitBreaker[http.Request, http.Response](c.maxRequests, c.interval, c.timeout, c.consecutiveFailures, resource)

	c.circuitBreakers[resource] = cb
	return cb
}

func getResource(r *http.Request) string {
	return r.Method + "_" + r.URL.Path
}
