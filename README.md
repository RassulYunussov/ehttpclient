# Enhanced Http Client
[![GoDoc](https://godoc.org/github.com/sony/gobreaker/v2?status.svg)](https://pkg.go.dev/github.com/RassulYunussov/ehttpclient)
![Build](https://github.com/rassulyunussov/ehttpclient/actions/workflows/go.yml/badge.svg)

Http Client Library that provides "enhanced" http client with resiliency & circuit breaker backed functionality

## Retry policy

EnhancedHttpClient will retry requests for 
- Network issues
- 5xx HTTP status
- Request Timeout

## Circuit breaker policy

Circuit breaker layer resides before retry policy layer and can be in __closed|open|half-open__ states. Used as-is from [gobreaker](https://github.com/sony/gobreaker)

Usage:

```
go get github.com/RassulYunussov/ehttpclient
```

By default ehttpclient.Create produces EnhancedHttpClient with no retry and no circuit breaker. Client can add policies:
- retry
- circuit breaker
- retry + cicruit breaker

### Default 

No retry policy. No circuit breaker

```
defaultClient := ehttpclient.Create(200*time.Millisecond)
```

### Retry

The retry policy uses multiplication of attempt & backoffTimeout

```
// retry count 3
// backoff timeout
retryClient := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithRetry(3, 3*100*time.Millisecond, 100*time.Millisecond))
```

### Circuit breaker

```
// detailed info about configuration can be found here: https://github.com/sony/gobreaker
retryClient := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithCircuitBreaker(1, 2, time.Second, time.Second))
```

### Retry + Circuit breaker

```
ehttpClient := ehttpclient.Create(200*time.Millisecond, ehttpclient.WithRetry(3, 3*100*time.Millisecond, 100*time.Millisecond), ehttpclient.WithCircuitBreaker(1, 2, time.Second, time.Second))
```

### Make a request
```
request, err := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
response, err := client.Do(request)
```

Notes: care should be taken with "Do" methoud using circuit breaker pattern. It uses resource URI. With high cardinality it may lead to memory exhaustion. Prefer using "DoResourceRequest" with controlled number of resources. 
