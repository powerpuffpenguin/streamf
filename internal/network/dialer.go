package network

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/powerpuffpenguin/vnet"
	"github.com/powerpuffpenguin/vnet/reverse"
)

type Dialer interface {
	DialContext(ctx context.Context) (conn net.Conn, e error)
	Close() error
}
type rawDialer struct {
	netDialer *net.Dialer
	network   string
	addr      string
	cfg       *tls.Config
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
func (d *rawDialer) Close() error {
	return nil
}

type pipeDialer struct {
	addr string
	cfg  *tls.Config
	pipe *vnet.PipeListener
	done chan struct{}
}

func (d *pipeDialer) DialContext(ctx context.Context) (conn net.Conn, e error) {
	select {
	case <-d.done:
	case <-ctx.Done():
		e = ctx.Err()
		return
	}

	conn, e = d.pipe.DialContext(ctx, `pipe`, d.addr)
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
func (d *pipeDialer) Close() error {
	return nil
}

type portalDialer struct {
	addr   string
	cfg    *tls.Config
	portal *reverse.Dialer
	done   chan struct{}
}

func (d *portalDialer) DialContext(ctx context.Context) (conn net.Conn, e error) {
	select {
	case <-d.done:
	case <-ctx.Done():
		e = ctx.Err()
		return
	}
	conn, e = d.portal.DialContext(ctx, `pipe`, d.addr)
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
func (d *portalDialer) Close() error {
	return nil
}
