package resilient

import "errors"

var ErrRetriesExhausted = errors.New("retries exhausted")
