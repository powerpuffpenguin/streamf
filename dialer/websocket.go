package dialer

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/sf/config"
	"github.com/powerpuffpenguin/sf/pool"
	"github.com/powerpuffpenguin/sf/third-party/websocket"
)

type websocketConn struct {
	ws *websocket.Conn
	r  io.Reader
}

func (w *websocketConn) Websocket() *websocket.Conn {
	return w.ws
}
func (w *websocketConn) Close() error {
	return w.ws.Close()
}
func (w *websocketConn) Write(b []byte) (n int, e error) {
	// panic(`not implemented websocketConn.Write`)
	e = w.ws.WriteMessage(websocket.BinaryMessage, b)
	if e != nil {
		n = len(b)
	}
	return
}
func (w *websocketConn) Read(b []byte) (n int, e error) {
	// panic(`not implemented websocketConn.Read`)
	r := w.r
	if r == nil {
		_, r, e = w.ws.NextReader()
		if e != nil {
			return
		}
		w.r = r
	}
	n, e = w.r.Read(b)
	if e == io.EOF {
		e = nil
		w.r = nil
	}
	return
}

type WebsocketDialer struct {
	done       chan struct{}
	clsoed     uint32
	remoteAddr RemoteAddr
	dialer     *websocket.Dialer
}

func newWebsocketDialer(log *slog.Logger, opts *config.Dialer, u *url.URL,
	secure bool,
	pool *pool.Pool,
) (dialer *WebsocketDialer, e error) {
	log = log.With(`dialer`, opts.Tag)
	var duration time.Duration
	if opts.Timeout == `` {
		duration = time.Millisecond * 500
	} else {
		var err error
		duration, err = time.ParseDuration(opts.Timeout)
		if err != nil {
			duration = time.Millisecond * 500
			log.Warn(`parse duration fail, used default close duration.`,
				`error`, err,
				`timeout`, duration,
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
	log.Info(`new dialer`,
		`network`, network,
		`addr`, addr,
		`url`, opts.URL,
		`timeout`, duration,
	)
	var netDialer net.Dialer
	dialer = &WebsocketDialer{
		done: make(chan struct{}),
		remoteAddr: RemoteAddr{
			Dialer:  opts.Tag,
			Network: network,
			Addr:    addr,
			Secure:  secure,
			URL:     opts.URL,
		},
		dialer: &websocket.Dialer{
			ReadBufferSize:  pool.Size(),
			WriteBufferSize: pool.Size(),
			WriteBufferPool: websocket.NewBufferPool(pool),
			NetDialContext: func(ctx context.Context, _, __ string) (net.Conn, error) {
				return netDialer.DialContext(ctx, network, addr)
			},
		},
	}
	if duration > 0 {
		dialer.dialer.HandshakeTimeout = duration
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

func (t *WebsocketDialer) Connect(ctx context.Context) (conn *Conn, e error) {
	ch := make(chan connectResult)
	go func() {
		conn, _, e := t.dialer.DialContext(ctx, t.remoteAddr.URL, nil)
		if e == nil {
			select {
			case ch <- connectResult{
				Conn: &Conn{
					ReadWriteCloser: &websocketConn{ws: conn},
					remoteAddr:      t.remoteAddr,
				},
			}:
			case <-t.done:
				conn.Close()
			case <-ctx.Done():
				conn.Close()
			}
		} else {
			select {
			case ch <- connectResult{
				Error: e,
			}:
			case <-t.done:
			case <-ctx.Done():
			}
		}
	}()
	select {
	case <-t.done:
		e = ErrClosed
	case <-ctx.Done():
		e = ctx.Err()
	case result := <-ch:
		conn, e = result.Conn, result.Error
	}
	return
}
