package dialer

import (
	"context"
	"log/slog"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"golang.org/x/net/proxy"
)

type SocksDialer struct {
	log        *slog.Logger
	done       chan struct{}
	closed     uint32
	remoteAddr RemoteAddr
	connect    string
	timeout    time.Duration
	retry      int
	rawDialer  proxy.ContextDialer
}

func newSocksDialer(log *slog.Logger, opts *config.Dialer, u *url.URL) (dialer *SocksDialer, e error) {
	log = log.With(`dialer`, opts.Tag)
	var timeout time.Duration
	if opts.Timeout == `` {
		timeout = time.Millisecond * 500
	} else {
		var err error
		timeout, err = time.ParseDuration(opts.Timeout)
		if err != nil {
			timeout = time.Millisecond * 500
			log.Warn(`parse duration fail, used default close duration.`,
				`error`, err,
				`timeout`, timeout,
			)
		}
	}
	var (
		network = `tcp`
		addr    = u.Host
		query   url.Values
	)
	if opts.Network != `` {
		network = opts.Network
	} else {
		query = u.Query()
		s := query.Get(`network`)
		if s != `` {
			network = s
		}
	}
	if opts.Addr != `` {
		addr = opts.Addr
	} else {
		if query == nil {
			query = u.Query()
		}
		s := query.Get(`addr`)
		if s != `` {
			addr = s
		}
	}
	var auth *proxy.Auth
	if opts.Socks.User != `` || opts.Socks.Password != `` {
		auth = &proxy.Auth{
			User:     opts.Socks.User,
			Password: opts.Socks.Password,
		}
	}
	rawDialer, e := proxy.SOCKS5(network, addr, auth, proxy.Direct)
	if e != nil {
		log.Error(`new dialer fail`, `error`, e)
		return
	}
	log.Info(`new dialer`,
		`network`, network,
		`addr`, addr,
		`url`, opts.URL,
		`timeout`, timeout,
		`connect`, opts.Socks.Connect,
	)
	dialer = &SocksDialer{
		log:  log,
		done: make(chan struct{}),
		remoteAddr: RemoteAddr{
			Dialer:  opts.Tag,
			Network: `tcp`,
			Addr:    addr,
			Secure:  false,
			URL:     opts.URL,
		},
		connect:   opts.Socks.Connect,
		timeout:   timeout,
		retry:     opts.Retry,
		rawDialer: rawDialer.(proxy.ContextDialer),
	}
	return
}
func (d *SocksDialer) Info() any {
	return map[string]any{
		`tag`:     d.remoteAddr.Dialer,
		`network`: d.remoteAddr.Network,
		`addr`:    d.remoteAddr.Addr,
		`url`:     d.remoteAddr.URL,

		`connect`: d.connect,

		`timeout`: d.timeout.String(),
		`retry`:   d.retry,
	}
}
func (d *SocksDialer) Tag() string {
	return d.remoteAddr.Dialer
}
func (d *SocksDialer) Close() (e error) {
	if d.closed == 0 && atomic.CompareAndSwapUint32(&d.closed, 0, 1) {
		close(d.done)
	} else {
		e = ErrClosed
	}
	return
}
func (d *SocksDialer) Connect(ctx context.Context) (conn *Conn, e error) {
	if d.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.timeout)
		defer cancel()
	}
	c, e := d.rawDialer.DialContext(ctx, `tcp`, d.connect)
	if e == nil {
		d.log.Debug(`socks connect success`)
		conn = &Conn{
			ReadWriteCloser: c,
			remoteAddr:      d.remoteAddr,
		}
	} else {
		d.log.Debug(`socks connect fail`, `error`, e)
	}
	return
}
