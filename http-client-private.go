package ehttpclient

import (
	"net/http"
	"time"
)

type enhancedHttpClientCreationParameters struct {
	retryParameters          *retryParameters
	circuitBreakerParameters *circuitBreakerParameters
}

type retryParameters struct {
	maxRetry       uint8
	backoffTimeout time.Duration
}

type circuitBreakerParameters struct {
	maxRequests         uint32
	consecutiveFailures uint32
	interval            time.Duration
	timeout             time.Duration
}

func getResource(r *http.Request) string {
	return r.Method + "_" + r.URL.Path
}
