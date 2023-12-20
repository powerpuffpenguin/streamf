package listener

import (
	"crypto/tls"
	"log/slog"
	"net"
	"sync/atomic"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/vnet/reverse"
)

type PortalListener struct {
	dialer *reverse.Dialer
	closed uint32
	log    *slog.Logger
}

func NewPortalListener(nk *network.Network,
	log *slog.Logger,
	opts *config.BasicListener, portal *config.Portal,
) (listener *PortalListener, e error) {
	secure, certificate, alpn, e := opts.TLS.Certificate()
	if e != nil {
		log.Error(`new portal listener fail`, `error`, e)
		return
	}
	var l net.Listener
	if secure {
		l, e = nk.ListenTLS(opts.Network, opts.Addr, &tls.Config{
			Certificates: []tls.Certificate{certificate},
			NextProtos:   alpn,
		})
		if e != nil {
			log.Error(`new portal listener fail`, `error`, e)
			return
		}
	} else {
		l, e = nk.Listen(opts.Network, opts.Addr)
		if e != nil {
			log.Error(`new portal listener fail`, `error`, e)
			return
		}
	}

	addr := l.Addr()
	tag := opts.Tag
	if tag == `` {
		if secure {
			tag = `portal ` + addr.Network() + `+tls://` + addr.String()
		} else {
			tag = `portal ` + addr.Network() + `://` + addr.String()
		}
	}
	log = log.With(`listener`, tag)
	if portal.Tag == `` {
		portal.Tag = tag
	}
	dialer, e := nk.NewPortal(log, l, portal)
	if e != nil {
		log.Error(`new portal listener fail`, `error`, e)
		return
	}
	listener = &PortalListener{
		dialer: dialer,
		log:    log,
	}
	return
}

func (l *PortalListener) Close() (e error) {
	if l.closed == 0 && atomic.CompareAndSwapUint32(&l.closed, 0, 1) {
		e = l.dialer.Close()
	} else {
		e = ErrClosed
	}
	return
}

func (l *PortalListener) Serve() error {
	return l.dialer.Serve()
}
