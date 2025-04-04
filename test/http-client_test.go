package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/RassulYunussov/ehttpclient"
	"github.com/RassulYunussov/ehttpclient/internal/resilient"
	"github.com/sony/gobreaker/v2"
	"gotest.tools/v3/assert"
)

const httpServerSleepTime = 50

type testServerConfig struct {
	returnStatus   int
	resource       string
	isTimeoutError bool
}

func createHttpServerWithConfigs(config []testServerConfig) (*httptest.Server, []*int) {
	calls := make([]*int, len(config))
	for i := 0; i < len(config); i++ {
		calls[i] = new(int)
	}
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for i := 0; i < len(config); i++ {
				if r.RequestURI == config[i].resource {
					*calls[i]++
					if config[i].isTimeoutError {
						time.Sleep(httpServerSleepTime * time.Millisecond)
					}
					w.WriteHeader(config[i].returnStatus)
					break
				}
			}
		}),
	), calls
}

func createHttpServer(returnStatus int, isTimeoutError bool) (*httptest.Server, *int) {
	server, calls := createHttpServerWithConfigs([]testServerConfig{{returnStatus, "/", isTimeoutError}})
	return server, calls[0]
}

func TestOk(t *testing.T) {
	s, calls := createHttpServer(http.StatusOK, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithRetry(3, 3*50*time.Millisecond, 50*time.Millisecond))
	resp, err := client.Do(request)
	assert.NilError(t, err)
	assert.Equal(t, 1, *calls, "expected 1 call")
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func TestNoResiliencyFeaturesOk(t *testing.T) {
	s, calls := createHttpServer(http.StatusOK, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200 * time.Millisecond)
	resp, err := client.Do(request)
	assert.NilError(t, err)
	assert.Equal(t, 1, *calls, "expected 1 call")
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func TestNoResiliencyFeatures5xxError(t *testing.T) {
	s, calls := createHttpServer(http.StatusInternalServerError, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200 * time.Millisecond)
	resp, err := client.Do(request)
	assert.NilError(t, err)
	assert.Equal(t, 1, *calls, "expected 1 call")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "expected http-500")
}

func TestNumberOfRequestsIs4For5xx(t *testing.T) {
	s, calls := createHttpServer(http.StatusInternalServerError, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(400*time.Millisecond, ehttpclient.WithRetry(3, 3*150*time.Millisecond, 30*time.Millisecond))
	_, err := client.Do(request)
	assert.ErrorIs(t, err, resilient.ErrRetriesExhausted)
	assert.Equal(t, 4, *calls, "expected 4 calls")
}

func TestNumberOfRequestsIs1For4xx(t *testing.T) {
	s, calls := createHttpServer(http.StatusBadRequest, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(400*time.Millisecond, ehttpclient.WithRetry(3, 3*30*time.Millisecond, 30*time.Millisecond))
	resp, err := client.Do(request)
	assert.NilError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, 1, *calls, "expected 1 call1")
}

func TestNumberOfRequestsIs1For5xxAndZeroRetry(t *testing.T) {
	s, calls := createHttpServer(http.StatusInternalServerError, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(10*time.Millisecond, ehttpclient.WithRetry(0, 0, time.Millisecond))
	resp, err := client.Do(request)
	assert.NilError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "expected http-500")
	assert.Equal(t, 1, *calls, "expected 1 call")
}

func TestNumberOfRequestsIs256For5xx(t *testing.T) {
	s, calls := createHttpServer(http.StatusInternalServerError, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(1000*time.Millisecond, ehttpclient.WithRetry(255, 3*100*time.Millisecond, time.Microsecond))
	_, err := client.Do(request)
	assert.ErrorIs(t, err, resilient.ErrRetriesExhausted)
	assert.Equal(t, 256, *calls, "expected 256 calls")
}

func TestTimeoutError(t *testing.T) {
	s, calls := createHttpServer(http.StatusOK, true)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(20 * time.Millisecond)
	_, err := client.Do(request)
	assert.Equal(t, (err.(*url.Error)).Timeout(), true)
	assert.Equal(t, 1, *calls, "expected 1 calls")
}

func TestContextDeadlineError(t *testing.T) {
	s, calls := createHttpServer(http.StatusOK, true)
	defer s.Close()
	timedContext, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	request, _ := http.NewRequestWithContext(timedContext, http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithRetry(3, 3*50*time.Millisecond, 50*time.Millisecond))
	_, err := client.Do(request)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 1, *calls, "expected 1 call")
}

func TestContextCancelError(t *testing.T) {
	s, calls := createHttpServer(http.StatusOK, true)
	defer s.Close()
	timedContext, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	cancel()
	request, _ := http.NewRequestWithContext(timedContext, http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithRetry(3, 3*time.Second, time.Second))
	_, err := client.Do(request)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, *calls, "expected 0 call")
}

func TestCircuitBreaker(t *testing.T) {
	s, calls := createHttpServer(http.StatusInternalServerError, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithCircuitBreaker(1, 1, time.Second, time.Second))
	resp, err1 := client.Do(request)
	assert.NilError(t, err1)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "expected http-500")
	for i := 0; i < 10; i++ {
		_, err := client.Do(request)
		assert.ErrorIs(t, err, gobreaker.ErrOpenState)
	}
	assert.Equal(t, 1, *calls, "expected only 1 request to reach server")
}

func TestCircuitBreakerDoesNotCount4xx(t *testing.T) {
	s, calls := createHttpServer(http.StatusBadRequest, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithCircuitBreaker(1, 1, time.Second, time.Second))
	for i := 0; i < 10; i++ {
		resp, err := client.Do(request)
		assert.NilError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected http-400")
	}
	assert.Equal(t, 10, *calls, "expected all 10 requests to reach server")
}

func TestResourceDependentCircuitBreakers(t *testing.T) {
	s, calls := createHttpServerWithConfigs([]testServerConfig{
		{http.StatusInternalServerError, "/v1/resource_one", false},
		{http.StatusOK, "/v1/resource_two", false},
	})
	defer s.Close()
	request1, _ := http.NewRequest(http.MethodGet, s.URL+"/v1/resource_one", nil)
	request2, _ := http.NewRequest(http.MethodGet, s.URL+"/v1/resource_two", nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithCircuitBreaker(1, 1, time.Second, time.Second))
	resp1, err1 := client.Do(request1)
	resp2, err2 := client.Do(request2)
	assert.NilError(t, err1)
	assert.NilError(t, err2)
	assert.Equal(t, http.StatusInternalServerError, resp1.StatusCode, "expected http-500")
	assert.Equal(t, http.StatusOK, resp2.StatusCode, "expected http-200")
	for i := 0; i < 10; i++ {
		_, err1 := client.Do(request1)
		resp2, err2 := client.Do(request2)
		assert.ErrorIs(t, err1, gobreaker.ErrOpenState)
		assert.NilError(t, err2)
		assert.Equal(t, http.StatusOK, resp2.StatusCode, "expected http-200")
	}
	assert.Equal(t, 1, *calls[0], "expected only 1 request to reach server")
	assert.Equal(t, 11, *calls[1], "expected only 1 request to reach server")
}

func TestRetryWithCircuitBreaker(t *testing.T) {
	s, calls := createHttpServer(http.StatusInternalServerError, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithRetry(2, 2*30*time.Millisecond, 10*time.Millisecond), ehttpclient.WithCircuitBreaker(1, 2, time.Second, time.Second))
	_, err := client.Do(request)
	assert.ErrorIs(t, err, gobreaker.ErrOpenState)
	assert.Equal(t, 2, *calls, "expected only 2 requests to reach server")
}

func TestCircuitBreakerTransitionToClosed(t *testing.T) {
	s, calls := createHttpServer(http.StatusInternalServerError, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithCircuitBreaker(1, 2, time.Second, time.Millisecond*100))
	resp1, err1 := client.Do(request)
	resp2, err2 := client.Do(request)
	_, err3 := client.Do(request)
	assert.NilError(t, err1)
	assert.NilError(t, err2)
	assert.Equal(t, http.StatusInternalServerError, resp1.StatusCode, "expected http-500")
	assert.Equal(t, http.StatusInternalServerError, resp2.StatusCode, "expected http-500")
	assert.ErrorIs(t, err3, gobreaker.ErrOpenState)
	time.Sleep(time.Millisecond * 100)
	_, err4 := client.Do(request)
	assert.NilError(t, err4)
	assert.Equal(t, 3, *calls, "expected circuit breaker to be closed and 4th request to reach server")
}
