package bridge

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
	"github.com/powerpuffpenguin/vnet/reverse"
)

type Bridge interface {
	Close() (e error)
	Serve() (e error)
	Info() any
}

func New(nk *network.Network, log *slog.Logger, pool *pool.Pool, dialers map[string]dialer.Dialer, opts *config.Bridge) (b Bridge, e error) {
	u, e := url.ParseRequestURI(opts.URL)
	if e != nil {
		log.Error(`url invalid`, `url`, opts.URL)
		return
	}
	switch u.Scheme {
	case dialer.Http:
		b, e = newHttpBridge(nk, log, pool, dialers, opts, u, false)
	case dialer.HttpTls:
		b, e = newHttpBridge(nk, log, pool, dialers, opts, u, true)
	case dialer.Websocket:
		b, e = newWebsocketBridge(nk, log, pool, dialers, opts, u, false)
	case dialer.WebsocketTls:
		b, e = newWebsocketBridge(nk, log, pool, dialers, opts, u, true)
	case dialer.Basic:
		b, e = newBasicBridge(nk, log, pool, dialers, opts, u, false)
	case dialer.BasicTls:
		b, e = newBasicBridge(nk, log, pool, dialers, opts, u, true)
	default:
		e = errors.New(`url scheme not supported: ` + opts.URL)
		log.Error(`url scheme not supported`, `url`, opts.URL)
	}
	return
}

type bridge struct {
	tag, network, addr, url string

	done     chan struct{}
	closed   uint32
	log      *slog.Logger
	listener *reverse.Listener
	closer   io.Closer

	pool          *pool.Pool
	dialer        dialer.Dialer
	closeDuration time.Duration
}

func newBridge(log *slog.Logger, l *reverse.Listener, closer io.Closer,
	pool *pool.Pool,
	dialer dialer.Dialer, closeDuration time.Duration,
	tag, network, addr, url string,
) *bridge {
	return &bridge{
		done:     make(chan struct{}),
		log:      log,
		listener: l,
		closer:   closer,

		pool:          pool,
		dialer:        dialer,
		closeDuration: closeDuration,

		tag:     tag,
		network: network,
		addr:    addr,
		url:     url,
	}
}
func (b *bridge) Info() any {
	return map[string]any{
		`tag`:     b.tag,
		`network`: b.network,
		`addr`:    b.addr,
		`url`:     b.url,

		`close`:  b.closeDuration.String(),
		`dialer`: b.dialer.Tag(),
	}
}
func (b *bridge) Close() (e error) {
	if b.closed == 0 && atomic.CompareAndSwapUint32(&b.closed, 0, 1) {
		close(b.done)
		if b.closer != nil {
			b.closer.Close()
		}
		e = b.listener.Close()
	} else {
		e = ErrClosed
	}
	return
}
func (b *bridge) Serve() (e error) {
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		rw, err := b.listener.Accept()
		if err != nil {
			if b.closed != 0 && atomic.LoadUint32(&b.closed) != 0 {
				return ErrClosed
			}

			if tempDelay == 0 {
				tempDelay = 5 * time.Millisecond
			} else {
				tempDelay *= 2
			}
			if max := 1 * time.Second; tempDelay > max {
				tempDelay = max
			}
			b.log.Warn(`accept fail`,
				`error`, err,
				`retrying`, tempDelay,
			)
			time.Sleep(tempDelay)
			continue
		}
		go b.serve(rw)
	}
}
func (b *bridge) serve(rw io.ReadWriteCloser) {
	dst, e := b.dialer.Connect(context.Background())
	if e != nil {
		rw.Close()
		b.log.Warn(`connect fail`,
			`error`, e,
		)
		return
	}
	addr := dst.RemoteAddr()
	b.log.Info(`bridge`,
		`network`, addr.Network,
		`addr`, addr.Addr,
		`secure`, addr.Secure,
		`url`, addr.URL,
	)
	network.Bridging(rw, dst.ReadWriteCloser, b.pool, b.closeDuration)
}

type emptyAddress struct {
}

func (emptyAddress) Network() string {
	return `bridge`
}
func (emptyAddress) String() string {
	return ``
}
