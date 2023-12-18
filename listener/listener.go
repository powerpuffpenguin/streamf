package listener

import (
	"errors"
	"log/slog"

	"github.com/powerpuffpenguin/sf/config"
	"github.com/powerpuffpenguin/sf/dialer"
	"github.com/powerpuffpenguin/sf/internal/network"
	"github.com/powerpuffpenguin/sf/pool"
)

type Listener interface {
	Close() (e error)
	Serve() (e error)
}

const (
	Basic = `basic`
	Http  = `http`
)

func New(nk *network.Network, log *slog.Logger, pool *pool.Pool, dialers map[string]dialer.Dialer, opts *config.Listener) (l Listener, e error) {
	switch opts.Mode {
	case Basic, "":
		if found, ok := dialers[opts.Dialer]; ok {
			l, e = NewBasicListener(nk, log, pool, found, &opts.BasicListener)
		} else {
			e = errors.New(`dialer not found: ` + opts.Dialer)
			log.Error(`dialer not found`, `dialer`, opts.Dialer)
		}
	case Http:
		l, e = NewHttpListener(nk, log, pool, dialers, &opts.BasicListener, opts.Router)
	default:
		e = errors.New(`listener mode not supported: ` + opts.Mode)
		log.Error(`listener mode not supported`, `mode`, opts.Mode)
	}
	return
}
