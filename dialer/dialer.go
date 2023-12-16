package dialer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"

	"github.com/powerpuffpenguin/sf/config"
	"github.com/powerpuffpenguin/sf/pool"
)

const (
	Websocket    = `ws`
	WebsocketTls = `wss`
	Http         = `http`
	HttpTls      = `https`
	Tcp          = `tcp`
	TcpTls       = `tcp+tls`
)

type Dialer interface {
	Tag() string
	Connect(ctx context.Context) (conn *Conn, e error)
	Close() (e error)
}

func New(log *slog.Logger, pool *pool.Pool, opts *config.Dialer) (dialer Dialer, e error) {
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
	case Websocket:
		dialer, e = newWebsocketDialer(log, opts, u, false, pool)
	case WebsocketTls:
		dialer, e = newWebsocketDialer(log, opts, u, true, pool)
	case Http:
		dialer, e = newHttpDialer(log, opts, u, false)
	case HttpTls:
		dialer, e = newHttpDialer(log, opts, u, true)
	case Tcp:
		dialer, e = newTcpDialer(log, opts, u, false)
	case TcpTls:
		dialer, e = newTcpDialer(log, opts, u, true)
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
