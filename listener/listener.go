package listener

import (
	"errors"
	"log/slog"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
)

type Listener interface {
	Close() (e error)
	Serve() (e error)
}

const (
	Basic  = `basic`
	Http   = `http`
	Portal = `portal`
)

func New(nk *network.Network, log *slog.Logger, pool *pool.Pool, dialers map[string]dialer.Dialer, opts *config.Listener) (l Listener, e error) {
	switch opts.Mode {
	case Basic, "":
		if found, ok := dialers[opts.Dialer.Tag]; ok {
			l, e = NewBasicListener(nk, log, pool, found, &opts.Dialer, &opts.BasicListener)
		} else {
			e = errors.New(`dialer not found: ` + opts.Dialer.Tag)
			log.Error(`dialer not found`, `dialer`, opts.Dialer.Tag)
		}
	case Http:
		l, e = NewHttpListener(nk, log, pool, dialers, &opts.BasicListener, opts.Router)
	case Portal:
		l, e = NewPortalListener(nk, log, &opts.BasicListener, &opts.Portal)
	default:
		e = errors.New(`listener mode not supported: ` + opts.Mode)
		log.Error(`listener mode not supported`, `mode`, opts.Mode)
	}
	return
}
