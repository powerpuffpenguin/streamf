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
)

type TcpDialer struct {
	done       chan struct{}
	clsoed     uint32
	duration   time.Duration
	remoteAddr RemoteAddr
	dialer     interface {
		DialContext(context.Context, string, string) (net.Conn, error)
	}
	config *tls.Config
}

func newTcpDialer(log *slog.Logger, opts *config.Dialer, u *url.URL, secure bool) (dialer *TcpDialer, e error) {
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
	dialer = &TcpDialer{
		done:     make(chan struct{}),
		duration: duration,
		remoteAddr: RemoteAddr{
			Dialer:  opts.Tag,
			Network: network,
			Addr:    addr,
			Secure:  secure,
			URL:     opts.URL,
		},
		dialer: new(net.Dialer),
	}
	if secure {
		dialer.config = &tls.Config{
			ServerName:         u.Hostname(),
			InsecureSkipVerify: opts.AllowInsecure,
		}
	}
	return
}
func (t *TcpDialer) Tag() string {
	return t.remoteAddr.Dialer
}
func (t *TcpDialer) Close() (e error) {
	if t.clsoed == 0 && atomic.CompareAndSwapUint32(&t.clsoed, 0, 1) {
		close(t.done)
	} else {
		e = ErrClosed
	}
	return
}
func (t *TcpDialer) Connect(ctx context.Context) (conn *Conn, e error) {
	if t.duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.duration)
		defer cancel()
	}
	ch := make(chan connectResult)
	go func() {
		conn, e := t.dialer.DialContext(ctx, t.remoteAddr.Network, t.remoteAddr.Addr)
		if e == nil && t.config != nil {
			tlsConn := tls.Client(conn, t.config.Clone())
			e = tlsConn.HandshakeContext(ctx)
			if e == nil {
				conn = tlsConn
			} else {
				conn.Close()
			}
		}
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
