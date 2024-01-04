package ehttpclient

import (
	"errors"

	local_errors "github.com/RassulYunussov/ehttpclient/internal/errors"
)

func IsHttp5xxStatusError(err error) bool {
	return errors.Is(err, local_errors.ErrHttp5xxStatus)
}
