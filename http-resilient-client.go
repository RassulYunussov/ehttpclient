package ehttpclient

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type resilientHttpClient struct {
	client         *http.Client
	maxRetry       uint8
	backoffTimeout time.Duration
	backoffs       []int64
}

func (c *resilientHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	return c.doWithRetry(r)
}

func (c *resilientHttpClient) Do(r *http.Request) (*http.Response, error) {
	return c.doWithRetry(r)
}

func (c *resilientHttpClient) do(r *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusInternalServerError {
		return resp, nil
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil, errHttp5xxStatus
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
	for i := uint16(0); i <= uint16(c.maxRetry); i++ {
		resp, err = c.do(r)
		if err == nil {
			return resp, nil
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, err
		}
		c.backoff(i)
	}
	return nil, err
}
