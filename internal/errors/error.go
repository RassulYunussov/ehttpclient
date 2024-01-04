package errors

import "errors"

var ErrHttp5xxStatus = errors.New("5xx http status error")
