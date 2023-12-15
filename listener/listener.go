package listener

import (
	"errors"
	"log/slog"

	"github.com/powerpuffpenguin/sf/config"
	"github.com/powerpuffpenguin/sf/dialer"
)

type Listener interface {
	Close() (e error)
	Serve() (e error)
}

const (
	Basic = `basic`
	Http  = `http`
)

func New(log *slog.Logger, dialers map[string]dialer.Dialer, opts *config.Listener) (l Listener, e error) {
	switch opts.Mode {
	case Basic, "":
		if found, ok := dialers[opts.Dialer]; ok {
			l, e = NewBasicListener(log, found, &opts.BasicListener)
		} else {
			e = errors.New(`dialer not found: ` + opts.Dialer)
			log.Error(`dialer not found`, `dialer`, opts.Mode)
		}
	case Http:
		l, e = NewHttpListener(log, dialers, &opts.BasicListener)
	default:
		e = errors.New(`listener mode not supported: ` + opts.Mode)
		log.Error(`listener mode not supported`, `mode`, opts.Mode)
	}
	return
}
