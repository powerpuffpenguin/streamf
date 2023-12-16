package dialer

import (
	"errors"
	"net/http"
)

var ErrClosed = errors.New(`dialer already closed`)
var errTagEmpty = errors.New(`tag must not be empty`)
var errNetworkUnix = errors.New(`network unix only supported on linux`)
var errClosed = errors.New(`conn already closed`)
var errHttpMethod = errors.New(`method must be "` + http.MethodPost + `" or "`+http.MethodPut+`" or "`+http.MethodPatch+`"`)
