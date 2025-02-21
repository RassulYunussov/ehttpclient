package common

import "net/http"

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
