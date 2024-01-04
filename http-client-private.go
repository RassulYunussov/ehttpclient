package ehttpclient

import (
	"github.com/RassulYunussov/ehttpclient/internal/cb"
	"github.com/RassulYunussov/ehttpclient/internal/resilient"
)

type enhancedHttpClientCreationParameters struct {
	retryParameters          *resilient.RetryParameters
	circuitBreakerParameters *cb.CircuitBreakerParameters
}
