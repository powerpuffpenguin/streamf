package dialer

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/internal/httpmux"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
	"github.com/powerpuffpenguin/streamf/third-party/websocket"
)

type WebsocketDialer struct {
	log        *slog.Logger
	done       chan struct{}
	closed     uint32
	remoteAddr RemoteAddr
	timeout    time.Duration
	retry      int
	dialer     *websocket.Dialer
	fast       bool
	header     http.Header
	rawDialer  network.Dialer
}

func newWebsocketDialer(nk *network.Network, log *slog.Logger, opts *config.Dialer, u *url.URL,
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
			log.Warn(`parse duration fail, used default timeout duration.`,
				`error`, err,
				`timeout`, timeout,
			)
		}
	}
	var (
		network = `tcp`
		addr    string
	)
	if opts.Network != `` {
		network = opts.Network
	}
	if opts.Addr == `` {
		if u.Port() == `` {
			if secure {
				addr = u.Host + `:443`
			} else {
				addr = u.Host + `:80`
			}
		} else {
			addr = u.Host
		}
	} else {
		addr = opts.Addr
	}
	rawDialer, e := nk.Dialer(network, addr, nil)
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

	var header http.Header
	if len(opts.Header) != 0 {
		header = make(http.Header, len(opts.Header)+1)
		for k, vs := range opts.Header {
			for _, v := range vs {
				header.Add(k, v)
			}
		}
	}
	if opts.Access != `` {
		access := `Bearer ` + base64.RawURLEncoding.EncodeToString([]byte(opts.Access))
		if header == nil {
			header = http.Header{
				`Authorization`: []string{access},
			}
		} else {
			header.Set(`Authorization`, access)
		}
	}
	dialer = &WebsocketDialer{
		log:  log,
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
		fast:      opts.Fast,
		header:    header,
		rawDialer: rawDialer,
	}
	if secure {
		dialer.dialer.TLSClientConfig = &tls.Config{
			ServerName:         u.Hostname(),
			InsecureSkipVerify: opts.AllowInsecure,
		}
	}
	return
}
func (d *WebsocketDialer) Info() any {
	return map[string]any{
		`tag`:     d.remoteAddr.Dialer,
		`network`: d.remoteAddr.Network,
		`addr`:    d.remoteAddr.Addr,
		`url`:     d.remoteAddr.URL,
		`secure`:  d.remoteAddr.Secure,

		`fast`:    d.fast,
		`timeout`: d.timeout.String(),
		`retry`:   d.retry,
	}
}
func (d *WebsocketDialer) Tag() string {
	return d.remoteAddr.Dialer
}
func (d *WebsocketDialer) Close() (e error) {
	if d.closed == 0 && atomic.CompareAndSwapUint32(&d.closed, 0, 1) {
		close(d.done)
		e = d.rawDialer.Close()
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
			var result connectResult
			if d.fast {
				result = connectResult{
					Conn: &Conn{
						ReadWriteCloser: conn.NetConn(),
						remoteAddr:      d.remoteAddr,
					},
				}
			} else {
				result = connectResult{
					Conn: &Conn{
						ReadWriteCloser: httpmux.NewWebsocketConn(conn),
						remoteAddr:      d.remoteAddr,
					},
				}
			}
			select {
			case ch <- result:
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
	if e == nil {
		d.log.Debug(`websocket connect success`)
	} else {
		d.log.Debug(`websocket connect fail`, `error`, e)
	}
	return
}
func (d *WebsocketDialer) connect(ctx context.Context) (conn *websocket.Conn, e error) {
	for i := 0; ; i++ {
		conn, _, e = d.dialer.DialContext(ctx, d.remoteAddr.URL, d.header)
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
