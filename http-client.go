package ehttpclient

import (
	"net/http"
	"sync"
	"time"

	"github.com/sony/gobreaker"
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
	resilientHttpClient := resilientHttpClient{client: &http.Client{Timeout: timeout}} // default to not retry
	enhancedHttpClientCreationParameters := new(enhancedHttpClientCreationParameters)
	for _, o := range opts {
		enhancedHttpClientCreationParameters = o(enhancedHttpClientCreationParameters)
	}
	if enhancedHttpClientCreationParameters.retryParameters != nil {
		retryParameters := enhancedHttpClientCreationParameters.retryParameters
		resilientHttpClient.maxRetry = retryParameters.maxRetry
		resilientHttpClient.backoffTimeout = retryParameters.backoffTimeout
		resilientHttpClient.backoffs = make([]int64, uint16(retryParameters.maxRetry))
		int64BackoffTimeout := int64(retryParameters.backoffTimeout)
		for i := int64(0); i < int64(retryParameters.maxRetry); i++ {
			resilientHttpClient.backoffs[i] = (i + 1) * int64BackoffTimeout
		}
	}
	if enhancedHttpClientCreationParameters.circuitBreakerParameters != nil {
		circuitBreakerParameters := enhancedHttpClientCreationParameters.circuitBreakerParameters
		return &circuitBreakerBackedHttpClient{
			maxRequests:         circuitBreakerParameters.maxRequests,
			consecutiveFailures: circuitBreakerParameters.consecutiveFailures,
			interval:            circuitBreakerParameters.interval,
			timeout:             circuitBreakerParameters.timeout,
			Mutex:               sync.Mutex{},
			resilientHttpClient: &resilientHttpClient,
			circuitBreakers:     make(map[string]*gobreaker.CircuitBreaker),
		}
	}
	return &resilientHttpClient
}

// Apply retry policy to EnhancedHttpClient
func WithRetry(maxRetry uint8,
	backoffTimeout time.Duration) func(h *enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters {
	return func(h *enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters {
		retryParameters := new(retryParameters)
		retryParameters.backoffTimeout = backoffTimeout
		retryParameters.maxRetry = maxRetry
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
		circuitBreakerParameters := new(circuitBreakerParameters)
		circuitBreakerParameters.maxRequests = maxRequests
		circuitBreakerParameters.consecutiveFailures = consecutiveFailures
		circuitBreakerParameters.interval = interval
		circuitBreakerParameters.timeout = timeout
		h.circuitBreakerParameters = circuitBreakerParameters
		return h
	}
}
