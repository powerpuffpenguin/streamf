package forwarding

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/sf/config"
)

var ErrListenerClosed = errors.New("listener already closed")

type Listener interface {
	Close() (e error)
	Serve() (e error)
}

func NewListener(log *slog.Logger, dialer map[string]Dialer, opts *config.Listener) (l Listener, e error) {
	switch opts.Mode {
	case "basic", "":
		if found, ok := dialer[opts.Dialer]; ok {
			l, e = NewBasicListener(log, found, &opts.BasicListener)
		} else {
			e = errors.New(`dialer not found: ` + opts.Dialer)
			log.Error(`dialer not found`, `dialer`, opts.Mode)
		}
	case "http":
		l, e = NewHttpListener(log, dialer, &opts.BasicListener)
	default:
		e = errors.New(`listener mode not supported: ` + opts.Mode)
		log.Error(`listener mode not supported`, `mode`, opts.Mode)
	}
	return
}

type BasicListener struct {
	listener net.Listener
	dialer   Dialer
	log      *slog.Logger
	closed   uint32
	duration time.Duration
}

func NewBasicListener(log *slog.Logger, dialer Dialer, opts *config.BasicListener) (listener *BasicListener, e error) {
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
			tag = `basic ` + addr.Network() + `tls://` + addr.String()
		} else {
			tag = `basic ` + addr.Network() + `://` + addr.String()
		}
	}
	log = log.With(`listener`, tag)
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
		log:      log,
		duration: duration,
	}
	return
}
func (l *BasicListener) Close() (e error) {
	if l.closed != 0 && atomic.CompareAndSwapUint32(&l.closed, 0, 1) {
		e = l.listener.Close()
	} else {
		e = ErrListenerClosed
	}
	return
}
func (l *BasicListener) Serve() error {
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		rw, err := l.listener.Accept()
		if err != nil {
			if l.closed != 0 && atomic.LoadUint32(&l.closed) != 0 {
				return ErrListenerClosed
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
		return
	}
	addr := dst.RemoteAddr()
	l.log.Info("bridge",
		`network`, addr.Network,
		`addr`, addr.Addr,
		`secure`, addr.Secure,
		`url`, addr.URL,
	)
	bridging(src, dst, l.duration)
}

type HttpListener struct {
}

func NewHttpListener(log *slog.Logger, dialer map[string]Dialer, opts *config.BasicListener) (listener *HttpListener, e error) {
	return
}
func (l *HttpListener) Close() (e error) {
	return
}
func (l *HttpListener) Serve() (e error) {
	return
}
