package bridge

import (
	"errors"
	"net/http"
)

var ErrClosed = errors.New(`bridge already closed`)
var errHttpMethod = errors.New(`method must be "` + http.MethodPost + `" or "` + http.MethodPut + `" or "` + http.MethodPatch + `"`)
