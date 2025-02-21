package ehttpclient

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

func getHttpServer(isOk, isTimeoutError bool) (*httptest.Server, *int) {
	calls := 0
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls++
			if isOk {
				w.WriteHeader(http.StatusOK)
			}
			if isTimeoutError {
				time.Sleep(httpServerSleepTime * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}),
	), &calls
}

func TestOk(t *testing.T) {
	s, calls := getHttpServer(true, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithRetry(3, 50*time.Millisecond))
	resp, err := client.Do(request)
	assert.NilError(t, err)
	assert.Equal(t, 1, *calls, "expected 1 call")
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func TestNoResiliencyFeaturesOk(t *testing.T) {
	s, calls := getHttpServer(true, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200 * time.Millisecond)
	resp, err := client.Do(request)
	assert.NilError(t, err)
	assert.Equal(t, 1, *calls, "expected 1 call")
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func TestNoResiliencyFeatures5xxError(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200 * time.Millisecond)
	resp, err := client.Do(request)
	assert.NilError(t, err)
	assert.Equal(t, 1, *calls, "expected 1 call")
	assert.Equal(t, 500, resp.StatusCode, "expected http-500")
}

func TestNumberOfRequestsIs4For5xx(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(400*time.Millisecond, ehttpclient.WithRetry(3, 30*time.Millisecond))
	_, err := client.Do(request)
	assert.ErrorIs(t, err, resilient.ErrRetriesExhausted)
	assert.Equal(t, 4, *calls, "expected 4 calls")
}

func TestNumberOfRequestsIs1For5xx(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(10*time.Millisecond, ehttpclient.WithRetry(0, time.Millisecond))
	resp, err := client.Do(request)
	assert.NilError(t, err)
	assert.Equal(t, 500, resp.StatusCode, "expected http-500")
	assert.Equal(t, 1, *calls, "expected 1 call")
}

func TestNumberOfRequestsIs256For5xx(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(1000*time.Millisecond, ehttpclient.WithRetry(255, time.Microsecond))
	_, err := client.Do(request)
	assert.ErrorIs(t, err, resilient.ErrRetriesExhausted)
	assert.Equal(t, 256, *calls, "expected 256 calls")
}

func TestTimeoutError(t *testing.T) {
	s, calls := getHttpServer(false, true)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(20 * time.Millisecond)
	_, err := client.Do(request)
	assert.Equal(t, (err.(*url.Error)).Timeout(), true)
	assert.Equal(t, 1, *calls, "expected 1 calls")
}

func TestContextDeadlineError(t *testing.T) {
	s, calls := getHttpServer(false, true)
	defer s.Close()
	timedContext, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	request, _ := http.NewRequestWithContext(timedContext, http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithRetry(3, 50*time.Millisecond))
	_, err := client.Do(request)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 1, *calls, "expected 1 call")
}

func TestContextCancelError(t *testing.T) {
	s, calls := getHttpServer(false, true)
	defer s.Close()
	timedContext, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	cancel()
	request, _ := http.NewRequestWithContext(timedContext, http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithRetry(3, time.Second))
	_, err := client.Do(request)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, *calls, "expected 0 call")
}

func TestCircuitBreaker(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithCircuitBreaker(1, 1, time.Second, time.Second))
	resp, err1 := client.Do(request)
	assert.NilError(t, err1)
	assert.Equal(t, 500, resp.StatusCode, "expected http-500")
	for i := 0; i < 10; i++ {
		_, err := client.Do(request)
		assert.ErrorIs(t, err, gobreaker.ErrOpenState)
	}
	assert.Equal(t, 1, *calls, "expected only 1 request to reach server")
}

func TestRetryWithCircuitBreaker(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithRetry(2, 10*time.Millisecond), ehttpclient.WithCircuitBreaker(1, 2, time.Second, time.Second))
	_, err1 := client.Do(request)
	assert.ErrorIs(t, err1, gobreaker.ErrOpenState)
	assert.Equal(t, 2, *calls, "expected only 2 requests to reach server")
}

func TestCircuitBreakerTransitionToClosed(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithCircuitBreaker(1, 2, time.Second, time.Millisecond*100))
	resp1, err1 := client.Do(request)
	resp2, err2 := client.Do(request)
	_, err3 := client.Do(request)
	assert.NilError(t, err1)
	assert.NilError(t, err2)
	assert.Equal(t, 500, resp1.StatusCode, "expected http-500")
	assert.Equal(t, 500, resp2.StatusCode, "expected http-500")
	assert.ErrorIs(t, err3, gobreaker.ErrOpenState)
	time.Sleep(time.Millisecond * 100)
	_, err4 := client.Do(request)
	assert.NilError(t, err4)
	assert.Equal(t, 3, *calls, "expected circuit breaker to be closed and 4th request to reach server")
}
