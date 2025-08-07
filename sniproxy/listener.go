package sniproxy

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
)

var ErrClosed = errors.New("listener already closed")

type Listener struct {
	listener net.Listener
	pool     *pool.Pool
	log      *slog.Logger
	closed   uint32
	close    chan struct{}

	timeout time.Duration

	tag, network, addr string

	accuracy map[string]accuracyMatcher
	order    []orderMatcher
	regexp   []regexpMatcher

	def, fallback                 dialer.Dialer
	defDuration, fallbackDuration time.Duration
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
		tag = `sniproxy ` + addr.Network() + `://` + addr.String()
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
		`sniff-timeout`, duration,
	)
	var (
		def         dialer.Dialer
		defDuration time.Duration
	)
	if opts.Default.Tag != `` {
		def = dialers[opts.Default.Tag]
		if def == nil {
			l.Close()
			e = errors.New(`dialer not found: ` + opts.Default.Tag)
			log.Error(`dialer not found`, `dialer`, opts.Default.Tag)
			return
		}
		defDuration, e = time.ParseDuration(opts.Default.Close)

		if e != nil {
			e = nil
			defDuration = time.Second
			log.Warn(`parse duration fail, used default close duration.`,
				`error`, e,
				`close`, opts.Default.Close,
				`default`, duration,
			)
		}
		log.Info(`sni default router`,
			`dialer`, def.Tag(),
			`close`, defDuration,
		)
	}
	var (
		fallback         dialer.Dialer
		fallbackDuration time.Duration
	)
	if opts.Fallback.Tag != `` {
		fallback = dialers[opts.Fallback.Tag]
		if fallback == nil {
			l.Close()
			e = errors.New(`dialer not found: ` + opts.Fallback.Tag)
			log.Error(`dialer not found`, `dialer`, opts.Fallback.Tag)
			return
		}
		fallbackDuration, e = time.ParseDuration(opts.Fallback.Close)
		if e != nil {
			e = nil
			fallbackDuration = time.Second
			log.Warn(`parse duration fail, used default close duration.`,
				`error`, e,
				`close`, opts.Default.Close,
				`default`, duration,
			)
		}
		log.Info(`sni fallback router`,
			`dialer`, fallback.Tag(),
			`close`, fallbackDuration,
		)
	}

	var (
		accuracy = make(map[string]accuracyMatcher)
		order    []orderMatcher
		reg      []regexpMatcher
	)
	for _, router := range opts.SNIRouter {
		dialer := dialers[router.Dialer.Tag]
		if dialer == nil {
			l.Close()
			e = errors.New(`dialer not found: ` + router.Dialer.Tag)
			log.Error(`dialer not found`, `dialer`, router.Dialer.Tag)
			return
		}
		duration, e = time.ParseDuration(router.Dialer.Close)
		if e != nil {
			e = nil
			duration = time.Second
			log.Warn(`parse duration fail, used default close duration.`,
				`error`, e,
				`close`, router.Dialer.Close,
				`default`, duration,
			)
		}

		for _, matcher := range router.Matcher {
			switch matcher.Type {
			default:
				if _, exists := accuracy[matcher.Value]; exists {
					l.Close()
					e = errors.New(`sni router repeat: ` + matcher.Value)
					log.Error(`sni accuracy fail`, `error`, e)
					return
				}
				accuracy[matcher.Value] = accuracyMatcher{
					dialer:   dialer,
					duration: duration,
				}
				log.Info(`sni accuracy`,
					`value`, matcher.Value,
					`dialer`, dialer.Tag(),
				)
			case `prefix`:
				order = append(order, orderMatcher{
					dialer:   dialer,
					duration: duration,
					prefix:   true,
					value:    matcher.Value,
				})
				log.Info(`sni prefix`,
					`value`, matcher.Value,
					`dialer`, dialer.Tag(),
				)
			case `suffix`:
				order = append(order, orderMatcher{
					dialer:   dialer,
					duration: duration,
					prefix:   false,
					value:    matcher.Value,
				})
				log.Info(`sni suffix`,
					`value`, matcher.Value,
					`dialer`, dialer.Tag(),
				)
			case `regexp`:
				r, err := regexp.Compile(matcher.Value)
				if err != nil {
					l.Close()
					e = err
					log.Error(`new regexp fail`,
						`error`, err,
						`value`, matcher.Value,
					)
					return
				}
				reg = append(reg, regexpMatcher{
					dialer:   dialer,
					duration: duration,
					value:    r,
				})
				log.Info(`sni regexp`,
					`value`, matcher.Value,
					`dialer`, dialer.Tag(),
				)
			}
		}
	}
	listener = &Listener{
		listener: l,
		pool:     pool,
		log:      log,

		timeout: duration,
		close:   make(chan struct{}),

		tag:     tag,
		network: addr.Network(),
		addr:    addr.String(),

		accuracy: accuracy,
		order:    order,
		regexp:   reg,

		def:              def,
		fallback:         fallback,
		defDuration:      defDuration,
		fallbackDuration: fallbackDuration,
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
		`sniff-timeout`: l.timeout,
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
func (l *Listener) serve(c net.Conn) {
	timer := time.NewTimer(l.timeout)
	var (
		serverName string
		sniBuffer  []byte
		sniClosed  bool
		sniError   error
		done       = make(chan struct{})
	)
	go func() {
		serverName, sniBuffer, sniClosed, sniError = sniffSNI(l.pool, c)
		close(done)
	}()

	select {
	case <-l.close:
		if !timer.Stop() {
			<-timer.C
		}
		c.Close()
		return
	case <-timer.C:
		l.log.Debug(`sniff timeout`)
		c.Close()
		return
	case <-done:
	}
	if sniError != nil {
		l.log.Warn(`get sni fail`, `error`, sniError)
		if !sniClosed && l.fallback != nil {
			dst, err := l.fallback.Connect(context.Background())
			if err != nil {
				l.log.Warn(`connect remote fail`, `error`, err)
				c.Close()
				return
			}
			l.log.Info(`sni bridging fallback`, `dialer`, l.fallback.Tag(), `remote`, dst.RemoteAddr().Addr)
			network.Bridging(&sniConn{
				Conn:   c,
				buffer: sniBuffer,
				pool:   l.pool,
			}, dst.ReadWriteCloser, l.pool, l.fallbackDuration)
			return
		}
		c.Close()
		return
	}

	// 優先匹配最精準的路由
	if matcher, ok := l.accuracy[serverName]; ok {
		dst, err := matcher.dialer.Connect(context.Background())
		if err != nil {
			l.log.Warn(`connect remote fail`, `error`, err)
			c.Close()
			return
		}
		l.log.Info(`sni bridging accuracy`, `dialer`, matcher.dialer.Tag(), `remote`, dst.RemoteAddr().Addr)
		network.Bridging(&sniConn{
			Conn:   c,
			buffer: sniBuffer,
			pool:   l.pool,
		}, dst.ReadWriteCloser, l.pool, matcher.duration)
		return
	}
	// 按順序匹配 前綴/後綴 路由
	for _, matcher := range l.order {
		if matcher.Match(serverName) {
			dst, err := matcher.dialer.Connect(context.Background())
			if err != nil {
				l.log.Warn(`connect remote fail`, `error`, err)
				c.Close()
				return
			}
			l.log.Info(`sni bridging order`, `dialer`, matcher.dialer.Tag(), `remote`, dst.RemoteAddr().Addr)
			network.Bridging(&sniConn{
				Conn:   c,
				buffer: sniBuffer,
				pool:   l.pool,
			}, dst.ReadWriteCloser, l.pool, matcher.duration)
			return
		}
	}
	// 最後匹配 最慢的 正則路由
	for _, matcher := range l.regexp {
		if matcher.Match(serverName) {
			dst, err := matcher.dialer.Connect(context.Background())
			if err != nil {
				l.log.Warn(`connect remote fail`, `error`, err)
				c.Close()
				return
			}
			l.log.Info(`sni bridging regexp`, `dialer`, matcher.dialer.Tag(), `remote`, dst.RemoteAddr().Addr)
			network.Bridging(&sniConn{
				Conn:   c,
				buffer: sniBuffer,
				pool:   l.pool,
			}, dst.ReadWriteCloser, l.pool, matcher.duration)
			return
		}
	}
	// 默認路由
	if l.def != nil {
		dst, err := l.def.Connect(context.Background())
		if err != nil {
			l.log.Warn(`connect remote fail`, `error`, err)
			c.Close()
			return
		}
		l.log.Info(`sni bridging default`, `dialer`, l.def.Tag(), `remote`, dst.RemoteAddr().Addr)
		network.Bridging(&sniConn{
			Conn:   c,
			buffer: sniBuffer,
			pool:   l.pool,
		}, dst.ReadWriteCloser, l.pool, l.defDuration)
		return
	}

	// 沒有匹配路由
	c.Close()
}

type sniConn struct {
	net.Conn
	buffer []byte
	n      int
	pool   *pool.Pool
}

func (s *sniConn) Read(b []byte) (int, error) {
	if s.buffer != nil {
		if len(b) == 0 {
			return s.Conn.Read(b)
		}
		n := copy(b, s.buffer[s.n:])
		s.n += n
		if s.n >= len(s.buffer) {
			if cap(s.buffer) == s.pool.Size() {
				s.pool.Put(s.buffer)
			}
			s.buffer = nil
		}
		return n, nil
	}
	return s.Conn.Read(b)
}
