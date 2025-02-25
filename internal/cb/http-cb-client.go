package cb

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/RassulYunussov/ehttpclient/internal/common"
)

type circuitBreakerBackedHttpClient struct {
	client              common.EnhancedHttpClient
	maxRequests         uint32
	consecutiveFailures uint32
	interval            time.Duration
	timeout             time.Duration
	circuitBreakers     sync.Map
}

func CreateCircuitBreakerHttpClient(client common.EnhancedHttpClient, circuitBreakerParameters *CircuitBreakerParameters) common.EnhancedHttpClient {
	return &circuitBreakerBackedHttpClient{
		maxRequests:         circuitBreakerParameters.MaxRequests,
		consecutiveFailures: circuitBreakerParameters.ConsecutiveFailures,
		interval:            circuitBreakerParameters.Interval,
		timeout:             circuitBreakerParameters.Timeout,
		client:              client,
	}
}

func (c *circuitBreakerBackedHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	cb := c.getCircuitBreaker(resource)
	resp, err := cb.execute(c.do, r)
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
	cb, _ := c.circuitBreakers.LoadOrStore(resource, newCircuitBreaker[http.Request, http.Response](c.maxRequests, c.interval, c.timeout, c.consecutiveFailures, resource))
	return cb.(*circuitBreaker[http.Request, http.Response])
}

func getResource(r *http.Request) string {
	return r.Method + "_" + r.URL.Path
}

func (c *circuitBreakerBackedHttpClient) do(r *http.Request) (*http.Response, error) {
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
