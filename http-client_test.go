package ehttpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/sony/gobreaker"
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
	client := CreateEnhancedHttpClient(200*time.Millisecond, WithRetry(3, 50))
	resp, err := client.Do(request)
	assert.NilError(t, err)
	assert.Equal(t, 1, *calls, "expected 1 call")
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func TestNumberOfRequestsIs4For5xx(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := CreateEnhancedHttpClient(200*time.Millisecond, WithRetry(3, 50))
	_, err := client.Do(request)
	assert.ErrorIs(t, err, ErrHttpStatus)
	assert.Equal(t, 4, *calls, "expected 4 calls")
}

func TestTimeoutError(t *testing.T) {
	s, calls := getHttpServer(false, true)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := CreateEnhancedHttpClient(20*time.Millisecond, WithRetry(3, 50))
	_, err := client.Do(request)
	assert.Equal(t, (err.(*url.Error)).Timeout(), true)
	assert.Equal(t, 4, *calls, "expected 4 calls")
}

func TestContextDeadlineError(t *testing.T) {
	s, calls := getHttpServer(false, true)
	defer s.Close()
	timedContext, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	request, _ := http.NewRequestWithContext(timedContext, http.MethodGet, s.URL, nil)
	client := CreateEnhancedHttpClient(200*time.Millisecond, WithRetry(3, 10))
	_, err := client.Do(request)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 1, *calls, "expected 1 call")
}

func TestCircuitBreaker(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := CreateEnhancedHttpClient(200*time.Millisecond, WithCircuitBreaker(1, 2, time.Second, time.Second))
	for i := 0; i < 3; i++ {
		_, err := client.Do(request)
		if i > 1 {
			assert.ErrorIs(t, err, gobreaker.ErrOpenState)
		} else {
			assert.ErrorIs(t, err, ErrHttpStatus)
		}
	}
	assert.Equal(t, 2, *calls, "expected only 2 requests to reach server")
}

func TestRetryWithCircuitBreaker(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := CreateEnhancedHttpClient(200*time.Millisecond, WithRetry(2, 10), WithCircuitBreaker(1, 2, time.Second, time.Second))
	for i := 0; i < 3; i++ {
		_, err := client.Do(request)
		if i > 1 {
			assert.ErrorIs(t, err, gobreaker.ErrOpenState)
		} else {
			assert.ErrorIs(t, err, ErrHttpStatus)
		}
	}
	assert.Equal(t, 6, *calls, "expected only 6 requests to reach server")
}

func TestCircuitBreakerTransitionToClosed(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := CreateEnhancedHttpClient(200*time.Millisecond, WithCircuitBreaker(1, 2, time.Second, time.Millisecond*100))
	for i := 0; i < 3; i++ {
		_, err := client.Do(request)
		if i > 1 {
			assert.ErrorIs(t, err, gobreaker.ErrOpenState)
		} else {
			assert.ErrorIs(t, err, ErrHttpStatus)
		}
	}
	time.Sleep(time.Millisecond * 100)
	_, err := client.Do(request)
	assert.ErrorIs(t, err, ErrHttpStatus)
	assert.Equal(t, 3, *calls, "expected circuit breaker to be closed and 3-d requests to reach server")
}
