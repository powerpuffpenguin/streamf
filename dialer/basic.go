package dialer

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/sf/config"
	"github.com/powerpuffpenguin/sf/internal/network"
)

type BasicDialer struct {
	done       chan struct{}
	clsoed     uint32
	remoteAddr RemoteAddr
	timeout    time.Duration
	retry      int
	rawDialer  network.Dialer
}

func newBasicDialer(nk *network.Network, log *slog.Logger, opts *config.Dialer, u *url.URL, secure bool) (dialer *BasicDialer, e error) {
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
	var cfg *tls.Config
	if secure {
		cfg = &tls.Config{
			ServerName:         u.Hostname(),
			InsecureSkipVerify: opts.AllowInsecure,
		}
	}
	rawDialer, e := nk.Dialer(network, addr, cfg)
	if e != nil {
		log.Error(`new dialer fail`, `error`, e)
		return
	}
	log.Info(`new dialer`,
		`network`, network,
		`addr`, addr,
		`url`, opts.URL,
		`timeout`, timeout,
	)
	dialer = &BasicDialer{
		done: make(chan struct{}),
		remoteAddr: RemoteAddr{
			Dialer:  opts.Tag,
			Network: network,
			Addr:    addr,
			Secure:  secure,
			URL:     opts.URL,
		},
		timeout:   timeout,
		retry:     opts.Retry,
		rawDialer: rawDialer,
	}
	return
}
func (d *BasicDialer) Tag() string {
	return d.remoteAddr.Dialer
}
func (d *BasicDialer) Close() (e error) {
	if d.clsoed == 0 && atomic.CompareAndSwapUint32(&d.clsoed, 0, 1) {
		close(d.done)
		e = d.rawDialer.Close()
	} else {
		e = ErrClosed
	}
	return
}
func (d *BasicDialer) Connect(ctx context.Context) (conn *Conn, e error) {
	if d.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.timeout)
		defer cancel()
	}
	ch := make(chan connectResult)
	go func() {
		conn, e := d.connect(ctx)
		if e == nil {
			select {
			case ch <- connectResult{
				Conn: &Conn{
					ReadWriteCloser: conn,
					remoteAddr:      d.remoteAddr,
				},
			}:
			case <-d.done:
				conn.Close()
			case <-ctx.Done():
				conn.Close()
			}
		} else {
			select {
			case ch <- connectResult{
				Error: e,
			}:
			case <-d.done:
			case <-ctx.Done():
			}
		}
	}()
	select {
	case <-d.done:
		e = ErrClosed
	case <-ctx.Done():
		e = ctx.Err()
	case result := <-ch:
		conn, e = result.Conn, result.Error
	}
	return
}
func (d *BasicDialer) connect(ctx context.Context) (conn net.Conn, e error) {
	for i := 0; ; i++ {
		conn, e = d.rawDialer.DialContext(ctx)
		if e == nil || i >= d.retry {
			break
		}
		select {
		case <-d.done:
			return
		case <-ctx.Done():
			return
		default:
		}
	}
	return
}
