package ehttpclient

import (
	"time"

	"github.com/RassulYunussov/ehttpclient/common"
	"github.com/RassulYunussov/ehttpclient/internal/cb"
	"github.com/RassulYunussov/ehttpclient/internal/resilient"
)

// Get new instance of EnhancedHttpClient
func Create(timeout time.Duration, opts ...func(*enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters) common.EnhancedHttpClient {
	enhancedHttpClientCreationParameters := new(enhancedHttpClientCreationParameters)
	for _, o := range opts {
		enhancedHttpClientCreationParameters = o(enhancedHttpClientCreationParameters)
	}
	circuitBreakerHttpClient := cb.CreateCircuitBreakerHttpClient(timeout, enhancedHttpClientCreationParameters.circuitBreakerParameters)
	if enhancedHttpClientCreationParameters.retryParameters != nil && enhancedHttpClientCreationParameters.retryParameters.MaxRetry > 0 {
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
