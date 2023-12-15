package dialer

import "errors"

var ErrClosed = errors.New("dialer already closed")
var errTagEmpty= errors.New(`tag must not be empty`)