# Enhanced Http Client

Http Client Library that provides "enhanced" http client with resiliency & circuit breaker backed functionality

More details: https://medium.com/@yunussov/enhanced-http-client-b406a8fa2c0b


Usage:

```
go get github.com/RassulYunussov/ehttpclient
```

```
client := ehttpclient.CreateEnhancedHttpClient(200*time.Millisecond, 3, 100, 10, 5)
request, err := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
response, err := client.Do(request)
```
