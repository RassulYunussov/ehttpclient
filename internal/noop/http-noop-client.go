package noop

import (
	"net/http"
	"time"

	"github.com/RassulYunussov/ehttpclient/common"
)

type noOpHttpClient struct {
	client *http.Client
}

func CreateNoOpHttpClient(timeout time.Duration) common.EnhancedHttpClient {
	return &noOpHttpClient{client: &http.Client{Timeout: timeout}}
}

func (c *noOpHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	return c.client.Do(r)
}

func (c *noOpHttpClient) Do(r *http.Request) (*http.Response, error) {
	return c.client.Do(r)
}
