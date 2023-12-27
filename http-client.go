package ehttpclient

import (
	"net/http"
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

// Enhanced HttpClient backed by resiliency patterns
// includes:
// - retry strategy
// - circuit breaker
type HttpClient interface {
	// method should have a semantic resource name that will be used to separate circuit breakers
	DoResourceRequest(resource string, r *http.Request) (*http.Response, error)
	// classic HttpClient interface support, gets resource from path + method
	Do(r *http.Request) (*http.Response, error)
}

// Get new instance of Enhanced Http Client
// Circuit breaker used github.com/sony/gobreaker
func CreateEnhancedHttpClient(timeout time.Duration,
	maxRetry uint8,
	backoffMs uint16,
	circuitBreakerMaxRequests uint32,
	circuitBreakerConsecutiveFailures uint32,
) HttpClient {
	resilientHttpClient := resilientHttpClient{
		client:    &http.Client{Timeout: timeout},
		maxRetry:  maxRetry,
		backoffMs: backoffMs,
	}
	return &circuitBreakerBackedHttpClient{
		MaxRequests:         circuitBreakerMaxRequests,
		ConsecutiveFailures: circuitBreakerConsecutiveFailures,
		Mutex:               sync.Mutex{},
		resilientHttpClient: &resilientHttpClient,
		circuitBreakers:     make(map[string]*gobreaker.CircuitBreaker),
	}
}

func getResource(r *http.Request) string {
	return r.Method + "_" + r.URL.Path
}
