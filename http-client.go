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
func CreateEnhancedHttpClient(timeout time.Duration, opts ...func(*enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters) HttpClient {
	resilientHttpClient := resilientHttpClient{
		client:   &http.Client{Timeout: timeout},
		maxRetry: 1, // default to not retry
	}
	enhancedHttpClientCreationParameters := new(enhancedHttpClientCreationParameters)
	for _, o := range opts {
		enhancedHttpClientCreationParameters = o(enhancedHttpClientCreationParameters)
	}

	if enhancedHttpClientCreationParameters.retryParameters != nil {
		retryParameters := enhancedHttpClientCreationParameters.retryParameters
		resilientHttpClient.maxRetry = retryParameters.maxRetry
		resilientHttpClient.backoffMs = retryParameters.backoffMs
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

type enhancedHttpClientCreationParameters struct {
	retryParameters          *retryParameters
	circuitBreakerParameters *circuitBreakerParameters
}

type retryParameters struct {
	maxRetry  uint8
	backoffMs uint16
}

type circuitBreakerParameters struct {
	maxRequests         uint32
	consecutiveFailures uint32
	interval            time.Duration
	timeout             time.Duration
}

func WithRetry(maxRetry uint8,
	backoffMs uint16) func(h *enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters {
	return func(h *enhancedHttpClientCreationParameters) *enhancedHttpClientCreationParameters {
		retryParameters := new(retryParameters)
		retryParameters.backoffMs = backoffMs
		retryParameters.maxRetry = maxRetry
		h.retryParameters = retryParameters
		return h
	}
}

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

func getResource(r *http.Request) string {
	return r.Method + "_" + r.URL.Path
}
