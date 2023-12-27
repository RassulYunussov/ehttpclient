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
	client := CreateEnhancedHttpClient(200*time.Millisecond, 3, 50, 50, 10, time.Second, time.Second)
	resp, err := client.Do(request)
	assert.NilError(t, err)
	assert.Equal(t, 1, *calls, "expected 1 call")
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func TestNumberOfRequestsIs3For5xx(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := CreateEnhancedHttpClient(200*time.Millisecond, 3, 50, 50, 10, time.Second, time.Second)
	_, err := client.Do(request)
	assert.ErrorIs(t, err, ErrHttpStatus)
	assert.Equal(t, 3, *calls, "expected 3 calls")
}

func TestTimeoutError(t *testing.T) {
	s, calls := getHttpServer(false, true)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := CreateEnhancedHttpClient(20*time.Millisecond, 3, 10, 50, 10, time.Second, time.Second)
	_, err := client.Do(request)
	assert.Equal(t, (err.(*url.Error)).Timeout(), true)
	assert.Equal(t, 3, *calls, "expected 3 calls")
}

func TestContextDeadlineError(t *testing.T) {
	s, calls := getHttpServer(false, true)
	defer s.Close()
	timedContext, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	request, _ := http.NewRequestWithContext(timedContext, http.MethodGet, s.URL, nil)
	client := CreateEnhancedHttpClient(20*time.Millisecond, 3, 10, 50, 10, time.Second, time.Second)
	_, err := client.Do(request)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 1, *calls, "expected 1 call")
}

func TestCircuitBreaker(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := CreateEnhancedHttpClient(200*time.Millisecond, 1, 10, 5, 2, time.Second, time.Second)
	for i := 0; i < 4; i++ {
		_, err := client.Do(request)
		if i > 2 {
			if i > 2 {
				assert.ErrorIs(t, err, gobreaker.ErrOpenState)
			} else {
				assert.ErrorIs(t, err, ErrHttpStatus)
			}
		}
	}
	assert.Equal(t, 3, *calls, "expected only 3 requests to reach server")
}

func TestCircuitBreakerTransitionToClosed(t *testing.T) {
	s, calls := getHttpServer(false, false)
	defer s.Close()
	request, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	client := CreateEnhancedHttpClient(200*time.Millisecond, 1, 10, 5, 2, time.Second, time.Millisecond*500)
	for i := 0; i < 4; i++ {
		_, err := client.Do(request)
		if i > 2 {
			if i > 2 {
				assert.ErrorIs(t, err, gobreaker.ErrOpenState)
			} else {
				assert.ErrorIs(t, err, ErrHttpStatus)
			}
		}
	}

	time.Sleep(time.Millisecond * 500)
	_, err := client.Do(request)
	assert.ErrorIs(t, err, ErrHttpStatus)
	assert.Equal(t, 4, *calls, "expected circuit breaker to close and 4-th requests to reach server")
}
