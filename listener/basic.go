package listener

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
)

type BasicListener struct {
	listener net.Listener
	dialer   dialer.Dialer
	pool     *pool.Pool
	log      *slog.Logger
	closed   uint32
	duration time.Duration

	tag, network, addr string
	secure             bool
}

func NewBasicListener(nk *network.Network, log *slog.Logger, pool *pool.Pool, dialer dialer.Dialer, connect *config.ConnectDialer, opts *config.BasicListener) (listener *BasicListener, e error) {
	secure, certificate, alpn, e := opts.TLS.Certificate()
	if e != nil {
		log.Error(`new basic listener fail`, `error`, e)
		return
	}
	var l net.Listener
	if secure {
		l, e = nk.ListenTLS(opts.Network, opts.Addr, &tls.Config{
			Certificates: []tls.Certificate{certificate},
			NextProtos:   alpn,
		})
		if e != nil {
			log.Error(`new basic listener fail`, `error`, e)
			return
		}
	} else {
		l, e = nk.Listen(opts.Network, opts.Addr)
		if e != nil {
			log.Error(`new basic listener fail`, `error`, e)
			return
		}
	}

	addr := l.Addr()
	tag := opts.Tag
	if tag == `` {
		if secure {
			tag = `basic ` + addr.Network() + `+tls://` + addr.String()
		} else {
			tag = `basic ` + addr.Network() + `://` + addr.String()
		}
	}
	log = log.With(`listener`, tag, `dialer`, dialer.Tag())
	var duration time.Duration
	if connect.Close == `` {
		duration = time.Second
	} else {
		var err error
		duration, err = time.ParseDuration(connect.Close)
		if err != nil {
			duration = time.Second
			log.Warn(`parse duration fail, used default close duration.`,
				`error`, err,
				`close`, connect.Close,
				`default`, duration,
			)
		}
	}
	log.Info(`new basic listener`, `close`, duration)
	listener = &BasicListener{
		listener: l,
		dialer:   dialer,
		pool:     pool,
		log:      log,
		duration: duration,

		tag:     tag,
		network: addr.Network(),
		addr:    addr.String(),
		secure:  secure,
	}
	return
}
func (l *BasicListener) Info() any {
	return map[string]any{
		`tag`:     l.tag,
		`network`: l.network,
		`addr`:    l.addr,
		`secure`:  l.secure,
		`dialer`:  l.dialer.Tag(),
		`portal`:  false,
	}
}
func (l *BasicListener) Close() (e error) {
	if l.closed == 0 && atomic.CompareAndSwapUint32(&l.closed, 0, 1) {
		e = l.listener.Close()
	} else {
		e = ErrClosed
	}
	return
}
func (l *BasicListener) Serve() error {
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		rw, err := l.listener.Accept()
		if err != nil {
			if l.closed != 0 && atomic.LoadUint32(&l.closed) != 0 {
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
			l.log.Warn(`basic accept fail`,
				`error`, err,
				`retrying`, tempDelay,
			)
			time.Sleep(tempDelay)
			continue
		}
		go l.serve(rw)
	}
}
func (l *BasicListener) serve(src net.Conn) {
	dst, e := l.dialer.Connect(context.Background())
	if e != nil {
		src.Close()
		l.log.Warn(`connect fail`, `error`, e)
		return
	}
	addr := dst.RemoteAddr()
	l.log.Info(`bridge`,
		`network`, addr.Network,
		`addr`, addr.Addr,
		`secure`, addr.Secure,
		`url`, addr.URL,
	)
	network.Bridging(src, dst.ReadWriteCloser, l.pool, l.duration)
}
