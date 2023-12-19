package bridge

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/url"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
	"github.com/powerpuffpenguin/vnet/reverse"
)

func newBasicBridge(nk *network.Network, log *slog.Logger, pool *pool.Pool, dialers map[string]dialer.Dialer, opts *config.Bridge, u *url.URL, secure bool) (bridge *bridge, e error) {
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
	var cfg *tls.Config
	if secure {
		cfg = &tls.Config{
			ServerName:         u.Hostname(),
			InsecureSkipVerify: opts.AllowInsecure,
		}
	}
	tag := opts.Tag
	if tag == `` {
		if secure {
			tag = `basic ` + network + `+tls://` + addr
		} else {
			tag = `basic ` + network + `://` + addr
		}
	}
	log = log.With(`bridge`, tag, `dialer`, opts.Dialer.Tag)
	rawDialer, e := nk.Dialer(network, addr, cfg)
	if e != nil {
		log.Error(`new dialer fail`, `error`, e)
		return
	}

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
	log.Info(`new bridge`,
		`network`, network,
		`addr`, addr,
		`url`, opts.URL,
	)
	l := reverse.Listen(emptyAddress{},
		reverse.WithListenerDialContext(func(ctx context.Context, network, address string) (net.Conn, error) {
			return rawDialer.DialContext(ctx)
		}),
		reverse.WithListenerSynAck(true),
	)
	bridge = newBridge(log, l, rawDialer,
		pool,
		found, closeDuration,
	)
	return
}
