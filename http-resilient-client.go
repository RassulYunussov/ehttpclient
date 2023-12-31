package ehttpclient

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"time"
)

type resilientHttpClient struct {
	client         *http.Client
	maxRetry       uint8
	backoffTimeout time.Duration
}

func (c *resilientHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	return c.doWithRetry(r)
}

func (c *resilientHttpClient) Do(r *http.Request) (*http.Response, error) {
	return c.DoResourceRequest(getResource(r), r)
}

func (c *resilientHttpClient) doWithRetry(r *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	for i := uint8(0); i <= c.maxRetry; i++ {
		resp, err = c.client.Do(r)
		if err == nil && resp.StatusCode < http.StatusInternalServerError {
			return resp, nil
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, err
		}
		if err == nil {
			err = ErrHttpStatus
		}
		if c.backoffTimeout > 1 {
			jitter := rand.Int63n(int64(c.backoffTimeout) / 2)
			backOff := int64(i+1) * int64(c.backoffTimeout)
			delay := time.Duration(backOff + jitter)
			time.Sleep(delay)
		}
	}
	return nil, err
}
