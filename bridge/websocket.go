package bridge

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/httpmux"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
	"github.com/powerpuffpenguin/streamf/third-party/websocket"
	"github.com/powerpuffpenguin/vnet/reverse"
)

func newWebsocketBridge(nk *network.Network, log *slog.Logger, pool *pool.Pool, dialers map[string]dialer.Dialer, opts *config.Bridge, u *url.URL, secure bool) (bridge *bridge, e error) {
	found, ok := dialers[opts.Dialer.Tag]
	if !ok {
		e = errors.New(`dialer not found: ` + opts.Dialer.Tag)
		log.Error(`dialer not found`, `dialer`, opts.Dialer.Tag)
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
	tag := opts.Tag
	if tag == `` {
		if secure {
			tag = `ws ` + network + `+tls://` + addr
		} else {
			tag = `ws ` + network + `://` + addr
		}
	}
	log = log.With(`bridge`, tag, `dialer`, opts.Dialer.Tag)
	rawDialer, e := nk.Dialer(network, addr, nil)
	if e != nil {
		log.Error(`new dialer fail`, `error`, e)
		return
	}
	var header http.Header
	if opts.Access != `` {
		access := `Bearer ` + base64.RawURLEncoding.EncodeToString([]byte(opts.Access))
		header = http.Header{
			`Authorization`: []string{access},
		}
	}
	websocketDialer := websocket.Dialer{
		ReadBufferSize:  pool.Size(),
		WriteBufferSize: pool.Size(),
		WriteBufferPool: websocket.NewBufferPool(pool),
		NetDialContext: func(ctx context.Context, _, __ string) (net.Conn, error) {
			return rawDialer.DialContext(ctx)
		},
	}
	if secure {
		websocketDialer.TLSClientConfig = &tls.Config{
			ServerName:         u.Hostname(),
			InsecureSkipVerify: opts.AllowInsecure,
		}
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
			ws, _, e := websocketDialer.DialContext(ctx, opts.URL, header)
			if e != nil {
				return nil, e
			}
			return httpmux.NewWebsocketConn(ws), nil
		}),
		reverse.WithListenerSynAck(true),
	)
	bridge = newBridge(log, l, rawDialer,
		pool,
		found, closeDuration,
	)
	return
}
