package dialer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"

	"github.com/powerpuffpenguin/sf/config"
)

const (
	Websocket    = `ws`
	WebsocketTls = `wss`
	Http         = `http`
	HttpTls      = `https`
	Tcp          = `tcp`
	TcpTls       = `tcp+tls`
	Unix         = `unix`
	UnixTls      = `unix+tls`
)

type Dialer interface {
	Connect(ctx context.Context) (conn *Conn, e error)
	Close() (e error)
}

func New(log *slog.Logger, opts *config.Dialer) (dialer Dialer, e error) {
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
	// case DialerWebsocket:
	// case DialerWebsocketTls:
	// case DialerHttp:
	// case DialerHttpTls:
	case Tcp:
		dialer, e = newTcpDialer(log, opts, u, false)
	case TcpTls:
		dialer, e = newTcpDialer(log, opts, u, true)
	case Unix:
		dialer, e = newUnixDialer(log, opts, u, false)
	case UnixTls:
		dialer, e = newUnixDialer(log, opts, u, true)
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
	Network string
	Addr    string
	Secure  bool
	URL     string
}
type connectResult struct {
	Conn  *Conn
	Error error
}
