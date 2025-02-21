package ehttpclient

import (
	"net/http"
	"time"

	"github.com/RassulYunussov/ehttpclient/internal/cb"
	"github.com/RassulYunussov/ehttpclient/internal/resilient"
)

// Enhanced HttpClient backed by resiliency patterns.
// Includes: retry & circuit breaker policies.
// Retriable errors: http-5xx, network errors
// Non-retriable errors: context.DeadlineExceeded|context.Canceled
type EnhancedHttpClient interface {
	// method should have a semantic resource name that will be used to separate circuit breakers
	DoResourceRequest(resource string, r *http.Request) (*http.Response, error)
	// classic HttpClient interface support, gets resource from path + method
	Do(r *http.Request) (*http.Response, error)
}

// Get new instance of EnhancedHttpClient
func Create(timeout time.Duration, opts ...func(*enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters) EnhancedHttpClient {
	enhancedHttpClientCreationParameters := new(enhancedHttpClientCreationParameters)
	for _, o := range opts {
		enhancedHttpClientCreationParameters = o(enhancedHttpClientCreationParameters)
	}
	circuitBreakerHttpClient := cb.CreateCircuitBreakerHttpClient(timeout, enhancedHttpClientCreationParameters.circuitBreakerParameters)
	if enhancedHttpClientCreationParameters.retryParameters != nil {
		return resilient.CreateResilientHttpClient(circuitBreakerHttpClient, timeout, enhancedHttpClientCreationParameters.retryParameters)
	}
	return circuitBreakerHttpClient
}

// Apply retry policy to EnhancedHttpClient
func WithRetry(maxRetry uint8,
	backoffTimeout time.Duration) func(h *enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters {
	return func(h *enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters {
		retryParameters := new(resilient.RetryParameters)
		retryParameters.BackoffTimeout = backoffTimeout
		retryParameters.MaxRetry = maxRetry
		h.retryParameters = retryParameters
		return h
	}
}

// Apply circuit breaker policy to EnhancedHttpClient.
// https://github.com/sony/gobreaker
func WithCircuitBreaker(maxRequests uint32,
	consecutiveFailures uint32,
	interval time.Duration,
	timeout time.Duration) func(h *enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters {
	return func(h *enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters {
		circuitBreakerParameters := new(cb.CircuitBreakerParameters)
		circuitBreakerParameters.MaxRequests = maxRequests
		circuitBreakerParameters.ConsecutiveFailures = consecutiveFailures
		circuitBreakerParameters.Interval = interval
		circuitBreakerParameters.Timeout = timeout
		h.circuitBreakerParameters = circuitBreakerParameters
		return h
	}
}
