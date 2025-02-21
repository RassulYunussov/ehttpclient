package resilient

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/RassulYunussov/ehttpclient/common"
	"github.com/sony/gobreaker"
)

type resilientHttpClient struct {
	client     common.EnhancedHttpClient
	maxRetry   uint8
	maxTimeout time.Duration
	backoffs   []int64
}

func CreateResilientHttpClient(enancedHttpClient common.EnhancedHttpClient, maxTimeout time.Duration, retryParameters *RetryParameters) common.EnhancedHttpClient {
	client := resilientHttpClient{client: enancedHttpClient} // default to not retry
	if retryParameters != nil {
		client.maxTimeout = maxTimeout
		client.maxRetry = retryParameters.MaxRetry
		client.backoffs = make([]int64, uint16(retryParameters.MaxRetry))
		int64BackoffTimeout := int64(retryParameters.BackoffTimeout)
		for i := int64(0); i < int64(retryParameters.MaxRetry); i++ {
			client.backoffs[i] = (i + 1) * int64BackoffTimeout
		}
	}
	return &client
}

func (c *resilientHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	return c.doWithRetry(r)
}

func (c *resilientHttpClient) Do(r *http.Request) (*http.Response, error) {
	return c.doWithRetry(r)
}

func (c *resilientHttpClient) backoff(step uint16) {
	if step != uint16(c.maxRetry) && c.backoffs[step] > 0 {
		jitter := rand.Int63n(c.backoffs[step] >> 1)
		delay := time.Duration(c.backoffs[step] + jitter)
		time.Sleep(delay)
	}
}

func (c *resilientHttpClient) doWithRetry(r *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	start := time.Now().UnixNano() / 1e6
	for i := uint16(0); i <= uint16(c.maxRetry); i++ {
		now := time.Now().UnixNano() / 1e6
		if c.maxTimeout.Milliseconds() <= now-start {
			return nil, context.DeadlineExceeded
		}
		resp, err = c.client.Do(r)
		if err == nil && resp.StatusCode < http.StatusInternalServerError {
			return resp, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, err
		}
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		c.backoff(i)
	}
	return nil, ErrRetriesExhausted
}
