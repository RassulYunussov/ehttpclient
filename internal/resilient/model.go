package resilient

import (
	"time"
)

type RetryParameters struct {
	MaxRetry       uint8
	MaxDelay       time.Duration
	BackoffTimeout time.Duration
}
