package sniproxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
)

var ErrClosed = errors.New("listener already closed")
var ErrGetSNI = errors.New("sniff sni fail")

type Listener struct {
	listener net.Listener
	pool     *pool.Pool
	log      *slog.Logger
	closed   uint32
	close    chan struct{}

	timeout time.Duration

	tag, network, addr string
}

func New(nk *network.Network, log *slog.Logger,
	pool *pool.Pool, dialers map[string]dialer.Dialer,
	opts *config.SNIProxy) (listener *Listener, e error) {

	l, e := nk.Listen(opts.Network, opts.Addr)
	if e != nil {
		log.Error(`new sniproxy listener fail`, `error`, e)
		return
	}
	addr := l.Addr()
	tag := opts.Tag
	if tag == `` {
		tag = `basic ` + addr.Network() + `://` + addr.String()
	}
	log = log.With(`sniproxy`, tag)

	var duration time.Duration
	if opts.Timeout == `` {
		duration = time.Millisecond * 500
	} else {
		var err error
		duration, err = time.ParseDuration(opts.Timeout)
		if err != nil {
			duration = time.Millisecond * 500
			log.Warn(`parse duration fail, used default sniff sni timeout duration.`,
				`error`, err,
				`close`, opts.Timeout,
				`default`, duration,
			)
		}
	}

	log.Info(`new sniproxy listener`,
		`network`, addr.Network(),
		`addr`, addr.String(),
		`sniff timeout`, duration,
	)

	listener = &Listener{
		listener: l,
		pool:     pool,
		log:      log,

		timeout: duration,
		close:   make(chan struct{}),

		tag:     tag,
		network: addr.Network(),
		addr:    addr.String(),
	}
	return
}
func (l *Listener) Close() (e error) {
	if l.closed == 0 && atomic.CompareAndSwapUint32(&l.closed, 0, 1) {
		close(l.close)
		e = l.listener.Close()
	} else {
		e = ErrClosed
	}
	return
}
func (l *Listener) Info() any {
	return map[string]any{
		`tag`:           l.tag,
		`network`:       l.network,
		`addr`:          l.addr,
		`sniff timeout`: l.timeout,
	}
}
func (l *Listener) Serve() (e error) {
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
			l.log.Warn(`sniproxy accept fail`,
				`error`, err,
				`retrying`, tempDelay,
			)
			time.Sleep(tempDelay)
			continue
		}
		go l.serve(rw)
	}
}
func (l *Listener) serve(src net.Conn) {
	timer := time.NewTimer(l.timeout)

	select {
	// case <-c.mu:
	// 	timer.Stop()
	case <-l.close:

	case <-timer.C:
		l.log.Debug(`sniff timeout`)
		src.Close()
		return
	}
	serverName, e := sniffSNI(src)
	fmt.Println(`-----`, serverName, e)
}

func sniffSNI(clientConn net.Conn) (string, error) {
	tlsConfig := &tls.Config{
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			return nil, fmt.Errorf("SNI: %s", info.ServerName)
		},
	}

	tlsConn := tls.Server(clientConn, tlsConfig)
	_ = tlsConn.Handshake()

	var serverName string
	err := tlsConn.Handshake()
	if err != nil {
		if tlsErr, ok := err.(net.Error); ok && tlsErr.Timeout() {
			return "", err
		}
		if err.Error()[:5] == "SNI: " {
			serverName = err.Error()[5:]
		}
	}

	if serverName == "" {
		return "", ErrGetSNI
	}

	return serverName, nil
}
