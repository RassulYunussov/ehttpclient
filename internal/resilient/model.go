package resilient

import (
	"time"
)

type RetryParameters struct {
	MaxRetry       uint8
	BackoffTimeout time.Duration
}
