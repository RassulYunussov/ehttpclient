# Enhanced Http Client

Http Client Library that provides "enhanced" http client with resiliency & circuit breaker backed functionality

More details: https://medium.com/@yunussov/enhanced-http-client-b406a8fa2c0b

Read more about circuit breaker configuration from: https://github.com/sony/gobreaker

Usage:

```
go get github.com/RassulYunussov/ehttpclient
```

By default ehttpclient.CreateEnhancedHttpClient produces standard HttpClient with no retry and no circuit breaker. Client can combine:
- retry
- circuit breaker
- retry + cicruit breaker


### Default 
```
defaultClient := ehttpclient.CreateEnhancedHttpClient(200*time.Millisecond)
```

### Retry

The retry policy uses multiplication of attempt & backoffMs. The result should not exceed uint16.
Reasonable values: 

- 1-5 for maxRetry
- 50 - 5000 milliseconds

```
// retry count 3
// backoff timeout range up to 100ms
retryClient := ehttpclient.CreateEnhancedHttpClient(200*time.Millisecond, ehttpclient.WithRetry(3, 100))
```

### Circuit breaker

```
// detailed info about configuration can be found here: https://github.com/sony/gobreaker
retryClient := ehttpclient.CreateEnhancedHttpClient(200*time.Millisecond, ehttpclient.WithCircuitBreaker(1, 2, time.Second, time.Second))
```

### Retry + Circuit breaker

```
ehttpClient := ehttpclient.CreateEnhancedHttpClient(200*time.Millisecond, ehttpclient.WithRetry(3, 100), ehttpclient.WithCircuitBreaker(1, 2, time.Second, time.Second))
```

### Make a request
```
request, err := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
response, err := client.Do(request)
```
