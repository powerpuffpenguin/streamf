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
	"github.com/powerpuffpenguin/sf/internal/httpmux"
	"github.com/powerpuffpenguin/sf/pool"
	"github.com/powerpuffpenguin/sf/third-party/websocket"
)

type WebsocketDialer struct {
	done       chan struct{}
	clsoed     uint32
	remoteAddr RemoteAddr
	timeout    time.Duration
	retry      int
	dialer     *websocket.Dialer
}

func newWebsocketDialer(log *slog.Logger, opts *config.Dialer, u *url.URL,
	secure bool,
	pool *pool.Pool,
) (dialer *WebsocketDialer, e error) {
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
	)
	if opts.Network != `` {
		network = opts.Network
	}
	if opts.Addr != `` {
		addr = opts.Addr
	}
	rawDialer, e := newRawDialer(network, addr, nil)
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
	dialer = &WebsocketDialer{
		done: make(chan struct{}),
		remoteAddr: RemoteAddr{
			Dialer:  opts.Tag,
			Network: network,
			Addr:    addr,
			Secure:  secure,
			URL:     opts.URL,
		},
		timeout: timeout,
		retry:   opts.Retry,
		dialer: &websocket.Dialer{
			ReadBufferSize:  pool.Size(),
			WriteBufferSize: pool.Size(),
			WriteBufferPool: websocket.NewBufferPool(pool),
			NetDialContext: func(ctx context.Context, _, __ string) (net.Conn, error) {
				return rawDialer.DialContext(ctx)
			},
		},
	}
	if secure {
		dialer.dialer.TLSClientConfig = &tls.Config{
			ServerName:         u.Hostname(),
			InsecureSkipVerify: opts.AllowInsecure,
		}
	}
	return
}

func (t *WebsocketDialer) Tag() string {
	return t.remoteAddr.Dialer
}
func (t *WebsocketDialer) Close() (e error) {
	if t.clsoed == 0 && atomic.CompareAndSwapUint32(&t.clsoed, 0, 1) {
		close(t.done)
	} else {
		e = ErrClosed
	}
	return
}

func (d *WebsocketDialer) Connect(ctx context.Context) (conn *Conn, e error) {
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
					ReadWriteCloser: httpmux.NewWebsocketConn(conn),
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
func (d *WebsocketDialer) connect(ctx context.Context) (conn *websocket.Conn, e error) {
	for i := 0; ; i++ {
		conn, _, e = d.dialer.DialContext(ctx, d.remoteAddr.URL, nil)
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
