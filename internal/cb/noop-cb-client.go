package cb

import "net/http"

type noCircuitBreakerHttpClient struct {
	client *http.Client
}

func (c *noCircuitBreakerHttpClient) DoResourceRequest(resource string, r *http.Request) (*http.Response, error) {
	return c.client.Do(r)
}

func (c *noCircuitBreakerHttpClient) Do(r *http.Request) (*http.Response, error) {
	return c.DoResourceRequest(getResource(r), r)
}
