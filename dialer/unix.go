package dialer

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/sf/config"
)

type UnixDialer struct {
	done       chan struct{}
	clsoed     uint32
	duration   time.Duration
	log        *slog.Logger
	host       string
	remoteAddr RemoteAddr
	dialer     interface {
		DialContext(context.Context, string, string) (net.Conn, error)
	}
}

func newUnixDialer(log *slog.Logger, opts *config.Dialer, u *url.URL, secure bool) (dialer *UnixDialer, e error) {
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
	var host string
	if opts.Addr == `` {
		host = u.Host
		rawURL := opts.URL[len(u.Scheme)+3:]
		if strings.HasPrefix(rawURL, `@`) {
			var (
				prefix string
				u      *url.URL
			)
			rawURL = rawURL[1:]
			if strings.HasPrefix(rawURL, `@`) {
				prefix = `XX`
				rawURL = rawURL[1:]
			} else {
				prefix = `X`
			}
			u, e = url.Parse(prefix + rawURL)
			if e != nil {
				log.Warn(`parse unix url fail`,
					`error`, e,
				)
				return
			}
			if prefix == `X` {
				host = `@` + u.Host
			} else {
				host = `@@` + u.Host
			}
		}
	}
	log.Info(`new dialer`,
		`url`, opts.URL,
		`timeout`, duration,
	)
	dialer = &UnixDialer{
		done:     make(chan struct{}),
		duration: duration,
		log:      log,
		host:     host,
		remoteAddr: RemoteAddr{
			Network: u.Scheme,
			Addr:    opts.Addr,
			Secure:  secure,
			URL:     opts.URL,
		},
	}
	if secure {
		dialer.dialer = &tls.Dialer{
			NetDialer: new(net.Dialer),
			Config: &tls.Config{
				ServerName:         u.Hostname(),
				InsecureSkipVerify: opts.AllowInsecure,
			},
		}
	} else {
		dialer.dialer = new(net.Dialer)
	}
	return
}
func (t *UnixDialer) Close() (e error) {
	if t.clsoed == 0 && atomic.CompareAndSwapUint32(&t.clsoed, 0, 1) {
		close(t.done)
	} else {
		e = ErrClosed
	}
	return
}
func (t *UnixDialer) Connect(ctx context.Context) (conn *Conn, e error) {
	if t.duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.duration)
		defer cancel()
	}
	ch := make(chan connectResult)
	go func() {
		conn, e := t.connect(ctx)
		if e == nil {
			select {
			case ch <- connectResult{
				Conn: &Conn{
					ReadWriteCloser: conn,
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
func (t *UnixDialer) connect(ctx context.Context) (conn net.Conn, e error) {
	if t.host == `` {
		conn, e = t.dialer.DialContext(ctx, `unix`, t.remoteAddr.Addr)
	} else {
		conn, e = t.dialer.DialContext(ctx, `unix`, t.host)
	}
	return
}
