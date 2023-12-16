package dialer

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/sf/config"
	"golang.org/x/net/http2"
)

type HttpDialer struct {
	done       chan struct{}
	clsoed     uint32
	remoteAddr RemoteAddr
	timeout    time.Duration
	retry      int
	client     *http.Client
	method     string
}

func newHttpDialer(log *slog.Logger, opts *config.Dialer, u *url.URL, secure bool) (dialer *HttpDialer, e error) {
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
	method := strings.ToUpper(opts.Method)
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		e = errHttpMethod
		log.Error(`http dialer method not supported`,
			`error`, e,
			`method`, opts.Method,
		)
		return
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

	var (
		rawDialer *rawDialer
		cfg       *tls.Config
	)
	if secure {
		cfg = &tls.Config{
			ServerName:         u.Hostname(),
			InsecureSkipVerify: opts.AllowInsecure,
		}
	}
	rawDialer, e = newRawDialer(network, addr, cfg)
	if e != nil {
		log.Error(`new dialer fail`, `error`, e)
		return
	}

	log.Info(`new dialer`,
		`network`, network,
		`addr`, addr,
		`url`, opts.URL,
		`timeout`, timeout,
		`method`, method,
	)
	dialer = &HttpDialer{
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
		client: &http.Client{
			Transport: &http2.Transport{
				AllowHTTP: !secure,
				DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
					return rawDialer.DialContext(ctx)
				},
			},
		},
		method: method,
	}
	return
}

func (d *HttpDialer) Tag() string {
	return d.remoteAddr.Dialer
}
func (d *HttpDialer) Close() (e error) {
	if d.clsoed == 0 && atomic.CompareAndSwapUint32(&d.clsoed, 0, 1) {
		close(d.done)
	} else {
		e = ErrClosed
	}
	return
}

func (d *HttpDialer) Connect(ctx context.Context) (conn *Conn, e error) {
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
func (d *HttpDialer) connect(ctx context.Context) (conn *httpConn, e error) {
	for i := 0; ; i++ {
		conn, e = d.connectHttp(ctx)
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
func (d *HttpDialer) connectHttp(ctx context.Context) (conn *httpConn, e error) {
	r, w := io.Pipe()
	req, e := http.NewRequest(d.method, d.remoteAddr.URL, r)
	if e != nil {
		w.Close()
		return
	}
	resp, e := d.client.Do(req)
	if e != nil {
		w.Close()
		return
	} else if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusAccepted {
		defer w.Close()
		var b []byte
		if resp.Body != nil {
			b, _ = io.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
		}
		if len(b) == 0 {
			e = errors.New(resp.Status)
		} else {
			e = errors.New(resp.Status + ` ` + string(b))
		}
		return
	} else if resp.Body == nil {
		e = errors.New(`http body nil`)
		return
	}
	conn = &httpConn{
		w: w,
		r: resp.Body,
	}
	return
}

type httpConn struct {
	w      *io.PipeWriter
	r      io.ReadCloser
	closed uint32
}

func (c *httpConn) Close() (e error) {
	if c.closed == 0 && atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		c.w.Close()
		c.r.Close()
	} else {
		e = errClosed
	}
	return
}
func (c *httpConn) Write(b []byte) (int, error) {
	return c.w.Write(b)
}
func (c *httpConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}
