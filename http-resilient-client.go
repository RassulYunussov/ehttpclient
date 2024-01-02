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
	backOffs       []int64
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
	backoffTimeout := int64(c.backoffTimeout)
	for i := int64(0); i <= int64(c.maxRetry); i++ {
		resp, err = c.client.Do(r)
		if err == nil && resp.StatusCode < http.StatusInternalServerError {
			return resp, nil
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, err
		}
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			err = errHttp5xxStatus
		}
		if backoffTimeout > 1 {
			jitter := rand.Int63n(c.backOffs[i] >> 1)
			delay := time.Duration(c.backOffs[i] + jitter)
			time.Sleep(delay)
		}
	}
	return nil, err
}
