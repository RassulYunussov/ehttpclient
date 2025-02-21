package cb

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

type CircuitBreakerHttpClient interface {
	DoResourceRequest(resource string, r *http.Request) (*http.Response, error)
	Do(r *http.Request) (*http.Response, error)
}

type circuitBreakerBackedHttpClient struct {
	sync.Mutex
	client              *http.Client
	maxRequests         uint32
	consecutiveFailures uint32
	interval            time.Duration
	timeout             time.Duration
	circuitBreakers     map[string]*circuitBreaker[http.Request, http.Response]
}

func CreateCircuitBreakerHttpClient(timeout time.Duration, circuitBreakerParameters *CircuitBreakerParameters) CircuitBreakerHttpClient {
	if circuitBreakerParameters == nil {
		return &noCircuitBreakerHttpClient{
			client: &http.Client{Timeout: timeout},
		}
	}
	return &circuitBreakerBackedHttpClient{
		maxRequests:         circuitBreakerParameters.MaxRequests,
		consecutiveFailures: circuitBreakerParameters.ConsecutiveFailures,
		interval:            circuitBreakerParameters.Interval,
		timeout:             circuitBreakerParameters.Timeout,
		Mutex:               sync.Mutex{},
		client:              &http.Client{Timeout: timeout},
		circuitBreakers:     make(map[string]*circuitBreaker[http.Request, http.Response]),
	}
}

func (c *circuitBreakerBackedHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	cb := c.getCircuitBreaker(resource)
	resp, err := cb.execute(c.execute, r)
	var e *circuitBreakerErrorWrapper[*http.Response]
	if errors.As(err, &e) {
		return e.wrapped, nil
	}
	return resp, err
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

func (c *circuitBreakerBackedHttpClient) execute(r *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusInternalServerError {
		return resp, nil
	}
	return nil, &circuitBreakerErrorWrapper[*http.Response]{
		wrapped: resp,
	}
}
