package ehttpclient

import "errors"

var errHttp5xxStatus = errors.New("5xx http status error")

func IsHttp5xxStatusError(err error) bool {
	return errors.Is(err, errHttp5xxStatus)
}
