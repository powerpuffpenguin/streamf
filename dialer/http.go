package dialer

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/internal/httpmux"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"golang.org/x/net/http2"
)

type HttpDialer struct {
	log        *slog.Logger
	done       chan struct{}
	closed     uint32
	remoteAddr RemoteAddr
	timeout    time.Duration
	retry      int
	client     *http.Client
	method     string
	header     http.Header
	rawDialer  network.Dialer
}

func newHttpDialer(nk *network.Network, log *slog.Logger, opts *config.Dialer, u *url.URL, secure bool) (dialer *HttpDialer, e error) {
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
	var ping time.Duration
	if opts.Ping == `` {
		ping = time.Second * 40
	} else {
		var err error
		ping, err = time.ParseDuration(opts.Ping)
		if err != nil {
			ping = time.Second * 40
			log.Warn(`parse duration fail, used default ping duration.`,
				`error`, err,
				`ping`, ping,
			)
		} else if ping < time.Second {
			ping = 0
		}
	}
	method := strings.ToUpper(opts.Method)
	switch method {
	case ``:
		method = http.MethodPost
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

	var (
		cfg *tls.Config
	)
	if secure {
		cfg = &tls.Config{
			NextProtos:         []string{`h2`},
			ServerName:         u.Hostname(),
			InsecureSkipVerify: opts.AllowInsecure,
		}
	}
	rawDialer, e := nk.Dialer(network, addr, cfg)
	if e != nil {
		log.Error(`new dialer fail`, `error`, e)
		return
	}

	log.Info(`new dialer`,
		`network`, network,
		`addr`, addr,
		`url`, opts.URL,
		`timeout`, timeout,
		`ping`, ping,
		`method`, method,
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
	dialer = &HttpDialer{
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
		client: &http.Client{
			Transport: &http2.Transport{
				ReadIdleTimeout: ping,
				AllowHTTP:       !secure,
				DialTLSContext: func(ctx context.Context, _, _ string, cfg *tls.Config) (net.Conn, error) {
					return rawDialer.DialContext(ctx)
				},
			},
		},
		method:    method,
		header:    header,
		rawDialer: rawDialer,
	}
	return
}
func (d *HttpDialer) Info() any {
	return map[string]any{
		`tag`:     d.remoteAddr.Dialer,
		`network`: d.remoteAddr.Network,
		`addr`:    d.remoteAddr.Addr,
		`url`:     d.remoteAddr.URL,
		`secure`:  d.remoteAddr.Secure,

		`close`: d.timeout.String(),
		`retry`: d.retry,
	}
}
func (d *HttpDialer) Tag() string {
	return d.remoteAddr.Dialer
}
func (d *HttpDialer) Close() (e error) {
	if d.closed == 0 && atomic.CompareAndSwapUint32(&d.closed, 0, 1) {
		close(d.done)
		e = d.rawDialer.Close()
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
	if e == nil {
		d.log.Debug(`http connect success`)
	} else {
		d.log.Debug(`http connect fail`, `error`, e)
	}
	return
}
func (d *HttpDialer) connect(ctx context.Context) (conn io.ReadWriteCloser, e error) {
	for i := 0; ; i++ {
		// conn, e = d.connectHttp(ctx)
		conn, e = httpmux.ConnectHttp(ctx, d.client, d.method, d.remoteAddr.URL, d.header)
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
