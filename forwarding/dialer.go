package forwarding

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/sf/config"
)

var ErrDialerClosed = errors.New("dialer already closed")

const (
	DialerWebsocket    = `ws`
	DialerWebsocketTls = `wss`
	DialerHttp         = `http`
	DialerHttpTls      = `https`
	DialerTcp          = `tcp`
	DialerTcpTls       = `tcp+tls`
	DialerUnix         = `unix`
	DialerUnixTls      = `unix+tls`
)

type Conn struct {
	io.ReadWriteCloser
	remoteAddr RemoteAddr
}

func (c *Conn) RemoteAddr() RemoteAddr {
	return c.remoteAddr
}

type Dialer interface {
	Connect(ctx context.Context) (conn *Conn, e error)
	Close() (e error)
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

func NewDialer(log *slog.Logger, opts *config.Dialer) (dialer Dialer, e error) {
	if opts.Tag == `` {
		e = errors.New(`tag must not be empty`)
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
	case DialerTcp:
		dialer, e = newTcpDialer(log, opts, u, false)
	case DialerTcpTls:
		dialer, e = newTcpDialer(log, opts, u, true)
	// case DialerUnix:
	// case DialerUnixTls:
	default:
		e = errors.New(`url scheme not supported: ` + opts.URL)
		log.Error(`url scheme not supported`, `url`, opts.URL)
	}
	return
}

type tcpDialer struct {
	done          chan struct{}
	clsoed        uint32
	duration      time.Duration
	log           *slog.Logger
	host          string
	allowInsecure bool
	remoteAddr    RemoteAddr
	dialer        interface {
		DialContext(context.Context, string, string) (net.Conn, error)
	}
}

func newTcpDialer(log *slog.Logger, opts *config.Dialer, u *url.URL, secure bool) (dialer *tcpDialer, e error) {
	log = log.With(`dialer`, opts.Tag)
	var duration time.Duration
	if opts.Timeout == `` {
		duration = time.Millisecond * 500
	} else {
		var err error
		duration, err = time.ParseDuration(opts.Timeout)
		if err != nil {
			duration = time.Millisecond * 500
			log.Warn(`parse duration fail, used default close duration.`,
				`error`, err,
				`timeout`, duration,
			)
		}
	}
	log.Info(`new dialer`,
		`url`, opts.URL,
		`timeout`, duration,
	)

	dialer = &tcpDialer{
		done:          make(chan struct{}),
		duration:      duration,
		log:           log,
		host:          u.Host,
		allowInsecure: opts.AllowInsecure,
		remoteAddr: RemoteAddr{
			Network: `tcp`,
			Addr:    opts.Addr,
			Secure:  secure,
			URL:     opts.URL,
		},
	}
	if secure {
		dialer.dialer = &tls.Dialer{
			NetDialer: new(net.Dialer),
			Config: &tls.Config{
				ServerName:         u.Hostname(),
				InsecureSkipVerify: opts.AllowInsecure,
			},
		}
	} else {
		dialer.dialer = new(net.Dialer)
	}
	return
}
func (t *tcpDialer) Close() (e error) {
	if t.clsoed == 0 && atomic.CompareAndSwapUint32(&t.clsoed, 0, 1) {
		close(t.done)
	} else {
		e = ErrDialerClosed
	}
	return
}
func (t *tcpDialer) Connect(ctx context.Context) (conn *Conn, e error) {
	if t.duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.duration)
		defer cancel()
	}
	ch := make(chan connectResult, 1)
	go func() {
		conn, e := t.connect(ctx)
		if e == nil {
			ch <- connectResult{
				Conn: &Conn{
					ReadWriteCloser: conn,
					remoteAddr:      t.remoteAddr,
				},
				Error: e,
			}
		} else {
			ch <- connectResult{
				Error: e,
			}
		}
	}()
	select {
	case <-t.done:
		e = ErrDialerClosed
	case <-ctx.Done():
		e = ctx.Err()
	case result := <-ch:
		conn, e = result.Conn, result.Error
	}
	return
}
func (t *tcpDialer) connect(ctx context.Context) (conn net.Conn, e error) {
	if t.remoteAddr.Addr == `` {
		conn, e = t.dialer.DialContext(ctx, `tcp`, t.host)
	} else {
		conn, e = t.dialer.DialContext(ctx, `tcp`, t.remoteAddr.Addr)
	}
	return
}
