package dialer

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"runtime"
)

type rawDialer struct {
	netDialer *net.Dialer
	network   string
	addr      string
	cfg       *tls.Config
}

func newRawDialer(network string, addr string, cfg *tls.Config) (dialer *rawDialer, e error) {
	switch network {
	case `tcp`:
	case `unix`:
		if runtime.GOOS != `linux` {
			e = errNetworkUnix
			return
		}
	default:
		e = errors.New(`network not supported: ` + network)
		return
	}
	netDialer := &net.Dialer{}
	dialer = &rawDialer{
		netDialer: netDialer,
		network:   network,
		addr:      addr,
		cfg:       cfg,
	}
	return
}
func (d *rawDialer) DialContext(ctx context.Context) (conn net.Conn, e error) {
	conn, e = d.netDialer.DialContext(ctx, d.network, d.addr)
	if d.cfg == nil || e != nil {
		return
	}

	tlsConn := tls.Client(conn, d.cfg.Clone())
	e = tlsConn.HandshakeContext(ctx)
	if e == nil {
		conn = tlsConn
	} else {
		conn.Close()
		conn = nil
	}
	return
}
