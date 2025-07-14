package dialer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
)

const (
	Socks        = `socks`
	Websocket    = `ws`
	WebsocketTls = `wss`
	Http         = `http`
	HttpTls      = `https`
	Basic        = `basic`
	BasicTls     = `basic+tls`
)

type Dialer interface {
	Tag() string
	Connect(ctx context.Context) (conn *Conn, e error)
	Close() (e error)
	Info() any
}

func New(nk *network.Network, log *slog.Logger, pool *pool.Pool, opts *config.Dialer) (dialer Dialer, e error) {
	if opts.Tag == `` {
		e = errTagEmpty
		log.Error(`tag must not be empty`)
		return
	}
	u, e := url.ParseRequestURI(opts.URL)
	if e != nil {
		log.Error(`url invalid`, `url`, opts.URL)
		return
	}
	switch u.Scheme {
	case Socks:
		dialer, e = newSocksDialer(log, opts, u)
	case Websocket:
		dialer, e = newWebsocketDialer(nk, log, opts, u, false, pool)
	case WebsocketTls:
		dialer, e = newWebsocketDialer(nk, log, opts, u, true, pool)
	case Http:
		dialer, e = newHttpDialer(nk, log, opts, u, false)
	case HttpTls:
		dialer, e = newHttpDialer(nk, log, opts, u, true)
	case Basic:
		if opts.Network == `udp` || opts.Network == `udp4` || opts.Network == `udp6` {
			dialer, e = newUdpDialer(nk, log, opts, u, pool)
		} else {
			dialer, e = newBasicDialer(nk, log, opts, u, false)
		}
	case BasicTls:
		dialer, e = newBasicDialer(nk, log, opts, u, true)
	default:
		e = errors.New(`url scheme not supported: ` + opts.URL)
		log.Error(`url scheme not supported`, `url`, opts.URL)
	}
	return
}

type Conn struct {
	io.ReadWriteCloser
	remoteAddr RemoteAddr
}

func (c *Conn) RemoteAddr() RemoteAddr {
	return c.remoteAddr
}

type RemoteAddr struct {
	Dialer  string
	Network string
	Addr    string
	Secure  bool
	URL     string
}
type connectResult struct {
	Conn  *Conn
	Error error
}
