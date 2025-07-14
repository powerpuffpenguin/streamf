package bridge

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/url"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/pool"
	"github.com/powerpuffpenguin/vnet/reverse"
	"golang.org/x/net/proxy"
)

func newSocksBridge(log *slog.Logger, pool *pool.Pool, dialers map[string]dialer.Dialer, opts *config.Bridge, u *url.URL) (bridge *bridge, e error) {
	found, ok := dialers[opts.Dialer.Tag]
	if !ok {
		e = errors.New(`dialer not found: ` + opts.Dialer.Tag)
		log.Error(`dialer not found`, `dialer`, opts.Dialer.Tag)
		return
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
	var auth *proxy.Auth
	if opts.Socks.User != `` || opts.Socks.Password != `` {
		auth = &proxy.Auth{
			User:     opts.Socks.User,
			Password: opts.Socks.Password,
		}
	}
	tag := opts.Tag
	if tag == `` {
		tag = `socks ` + network + `://` + addr
	}
	log = log.With(`bridge`, tag, `dialer`, opts.Dialer.Tag)
	rawDialer, e := proxy.SOCKS5(network, addr, auth, proxy.Direct)
	if e != nil {
		log.Error(`new dialer fail`, `error`, e)
		return
	}
	socksDialer := rawDialer.(proxy.ContextDialer)
	connect := opts.Socks.Connect

	var closeDuration time.Duration
	if opts.Dialer.Close == `` {
		closeDuration = time.Second
	} else {
		var err error
		closeDuration, err = time.ParseDuration(opts.Dialer.Close)
		if err != nil {
			closeDuration = time.Second
			log.Warn(`parse duration fail, used default close duration.`,
				`error`, err,
				`close`, opts.Dialer.Close,
				`default`, closeDuration,
			)
		}
	}
	log.Info(`new basic bridge`,
		`network`, network,
		`addr`, addr,
		`url`, opts.URL,
	)
	l := reverse.Listen(emptyAddress{},
		reverse.WithListenerDialContext(func(ctx context.Context, network, address string) (net.Conn, error) {
			return socksDialer.DialContext(ctx, `tcp`, connect)
		}),
		reverse.WithListenerSynAck(true),
	)
	bridge = newBridge(log, l, nil,
		pool,
		found, closeDuration,
		tag, network, addr, opts.URL,
	)
	return
}
