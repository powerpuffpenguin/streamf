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
	"strings"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/httpmux"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
	"github.com/powerpuffpenguin/vnet/reverse"
	"golang.org/x/net/http2"
)

func newHttpBridge(nk *network.Network, log *slog.Logger, pool *pool.Pool, dialers map[string]dialer.Dialer, opts *config.Bridge, u *url.URL, secure bool) (bridge *bridge, e error) {
	found, ok := dialers[opts.Dialer.Tag]
	if !ok {
		e = errors.New(`dialer not found: ` + opts.Dialer.Tag)
		log.Error(`dialer not found`, `dialer`, opts.Dialer.Tag)
		return
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
		addr    = u.Host
	)
	if opts.Network != `` {
		network = opts.Network
	}
	if opts.Addr != `` {
		addr = opts.Addr
	}
	var cfg *tls.Config
	if secure {
		cfg = &tls.Config{
			NextProtos:         []string{`h2`},
			ServerName:         u.Hostname(),
			InsecureSkipVerify: opts.AllowInsecure,
		}
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
	rawDialer, e := nk.Dialer(network, addr, cfg)
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
	client := &http.Client{
		Transport: &http2.Transport{
			ReadIdleTimeout: ping,
			AllowHTTP:       !secure,
			DialTLSContext: func(ctx context.Context, _, _ string, cfg *tls.Config) (net.Conn, error) {
				return rawDialer.DialContext(ctx)
			},
		},
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
		`method`, method,
	)
	l := reverse.Listen(emptyAddress{},
		reverse.WithListenerDialContext(func(ctx context.Context, network, address string) (net.Conn, error) {
			return httpmux.ConnectHttp(context.Background(), client, method, opts.URL, header)
		}),
		reverse.WithListenerSynAck(true),
	)
	bridge = newBridge(log, l, rawDialer,
		pool,
		found, closeDuration,
	)
	return
}
