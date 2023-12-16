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
	log.Info(`new dialer`,
		`url`, opts.URL,
		`timeout`, duration,
	)
	var (
		network string
		addr    string
	)
	if opts.Network == `` {
		network = `tcp`
	} else {
		network = opts.Network
	}
	if opts.Addr == `` {
		addr = u.Host
	} else {
		addr = opts.Addr
	}
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
