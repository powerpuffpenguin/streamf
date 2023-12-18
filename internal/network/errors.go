package network

import "errors"

var ErrNetworkUnix = errors.New(`network unix only supported on linux`)
