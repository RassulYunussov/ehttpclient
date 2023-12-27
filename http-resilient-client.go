package ehttpclient

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"time"
)

var ErrHttpStatus = errors.New("5xx http status error")

type resilientHttpClient struct {
	client    *http.Client
	maxRetry  uint8
	backoffMs uint16
}

func (c *resilientHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	return c.doWithRetry(resource, r)
}

func (c *resilientHttpClient) Do(r *http.Request) (*http.Response, error) {
	return c.DoResourceRequest(getResource(r), r)
}

func (c *resilientHttpClient) doWithRetry(resource string, r *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	for i := uint8(0); i < c.maxRetry; i++ {
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
		jitter := uint16(rand.Intn(int(c.backoffMs / 2)))
		backOff := uint16((i + 1)) * c.backoffMs
		delay := time.Duration(backOff+jitter) * time.Millisecond
		time.Sleep(delay)
	}
	return nil, err
}
