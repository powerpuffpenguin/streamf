package listener

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/sf/config"
	"github.com/powerpuffpenguin/sf/dialer"
	"github.com/powerpuffpenguin/sf/pool"
)

type BasicListener struct {
	listener net.Listener
	dialer   dialer.Dialer
	pool     *pool.Pool
	log      *slog.Logger
	closed   uint32
	duration time.Duration
}

func NewBasicListener(log *slog.Logger, pool *pool.Pool, dialer dialer.Dialer, opts *config.BasicListener) (listener *BasicListener, e error) {
	var (
		l      net.Listener
		secure bool
	)
	if opts.CertFile != `` && opts.KeyFile != `` {
		secure = true
		var certificate tls.Certificate
		certificate, e = tls.LoadX509KeyPair(opts.CertFile, opts.KeyFile)
		if e != nil {
			log.Error(`new basic listener fail`, `error`, e)
			return
		}
		l, e = tls.Listen(opts.Network, opts.Address, &tls.Config{
			Certificates: []tls.Certificate{certificate},
		})
		if e != nil {
			log.Error(`new basic listener fail`, `error`, e)
			return
		}
	} else {
		l, e = net.Listen(opts.Network, opts.Address)
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
	if opts.Close == `` {
		duration = time.Second
	} else {
		var err error
		duration, err = time.ParseDuration(opts.Close)
		if err != nil {
			duration = time.Second
			log.Warn(`parse duration fail, used default close duration.`,
				`error`, err,
				`close`, duration,
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
	}
	return
}
func (l *BasicListener) Close() (e error) {
	if l.closed != 0 && atomic.CompareAndSwapUint32(&l.closed, 0, 1) {
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
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
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
			return err
		}
		go l.serve(rw)
	}
}
func (l *BasicListener) serve(src net.Conn) {
	defer src.Close()
	src.RemoteAddr()
	dst, e := l.dialer.Connect(context.Background())
	if e != nil {
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
	bridging(src, dst.ReadWriteCloser, l.pool, l.duration)
}
