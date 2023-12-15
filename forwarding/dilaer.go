package forwarding

import (
	"context"
	"net"
)

type Dialer interface {
	Connect(ctx context.Context) (conn net.Conn, addr *RemoteAddr, e error)
}
type RemoteAddr struct {
	Network string
	Addr    string
	Secure  bool
	URL     string
}
