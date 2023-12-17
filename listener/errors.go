package listener

import (
	"errors"
	"net/http"
)

var ErrClosed = errors.New("listener already closed")
var errHttpMethod = errors.New(`method must be "` + http.MethodPost + `" or "` + http.MethodPut + `" or "` + http.MethodPatch + `" or "WS"`)
