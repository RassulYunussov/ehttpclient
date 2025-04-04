package ehttpclient

import (
	"net/http"
	"time"

	"github.com/RassulYunussov/ehttpclient/internal/cb"
	"github.com/RassulYunussov/ehttpclient/internal/noop"
	"github.com/RassulYunussov/ehttpclient/internal/resilient"
)

// Enhanced HttpClient backed by resiliency patterns.
// Includes: retry & circuit breaker policies.
// Retriable errors: http-5xx, network errors
// Non-retriable errors: context.DeadlineExceeded|context.Canceled|gobreaker.ErrOpenState|gobreaker.ErrTooManyRequests
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
	client := noop.CreateNoOpHttpClient(timeout)
	if enhancedHttpClientCreationParameters.circuitBreakerParameters != nil {
		client = cb.CreateCircuitBreakerHttpClient(client, enhancedHttpClientCreationParameters.circuitBreakerParameters)
	}
	if enhancedHttpClientCreationParameters.retryParameters != nil && enhancedHttpClientCreationParameters.retryParameters.MaxRetry > 0 {
		client = resilient.CreateResilientHttpClient(client, enhancedHttpClientCreationParameters.retryParameters)
	}
	return client
}

// Apply retry policy to EnhancedHttpClient
func WithRetry(maxRetry uint8,
	maxDelay time.Duration,
	backoffTimeout time.Duration) func(h *enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters {
	return func(h *enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters {
		retryParameters := new(resilient.RetryParameters)
		retryParameters.BackoffTimeout = backoffTimeout
		retryParameters.MaxRetry = maxRetry
		retryParameters.MaxDelay = maxDelay
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
