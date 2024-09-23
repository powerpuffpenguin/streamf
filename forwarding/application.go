package forwarding

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/powerpuffpenguin/streamf/bridge"
	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/internal/udp"
	"github.com/powerpuffpenguin/streamf/listener"
	"github.com/powerpuffpenguin/streamf/pool"
)

type Application struct {
	bridges   []bridge.Bridge
	listeners []listener.Listener
	dialers   map[string]dialer.Dialer
	udps      []*udp.UDP
	log       *slog.Logger
}

func NewApplication(conf *config.Config) (app *Application, e error) {
	log, e := newLogger(&conf.Logger)
	if e != nil {
		return
	}
	app, bridges, dialers, listeners, udps, e := newApplication(log, conf)
	if e != nil {
		for _, d := range dialers {
			d.Close()
		}
		for _, b := range bridges {
			b.Close()
		}
		for _, l := range listeners {
			l.Close()
		}
		for _, u := range udps {
			u.Close()
		}
	}
	return
}
func newApplication(log *slog.Logger, conf *config.Config) (app *Application,
	bridges []bridge.Bridge,
	dialers map[string]dialer.Dialer,
	listeners []listener.Listener,
	udps []*udp.UDP,
	e error) {
	bridges = make([]bridge.Bridge, 0, len(conf.Bridge))
	dialers = make(map[string]dialer.Dialer, len(conf.Dialer))
	listeners = make([]listener.Listener, 0, len(conf.Listener))
	udps = make([]*udp.UDP, 0, len(conf.UDP))
	var (
		tag    string
		exists bool
		pool   = pool.New(&conf.Pool)
		nk     = network.New()
	)
	app = &Application{
		log: log,
	}
	api := app.api()
	var d dialer.Dialer
	for _, opts := range conf.Dialer {
		tag = opts.Tag
		if _, exists = dialers[tag]; exists {
			e = fmt.Errorf(`dialer tag repeat: %s`, opts.Tag)
			log.Error(`dialer tag repeat`, `tag`, opts.Tag)
			return
		}
		d, e = dialer.New(nk, log, pool, opts)
		if e != nil {
			return
		}
		dialers[opts.Tag] = d
	}
	var b bridge.Bridge
	for _, opts := range conf.Bridge {
		b, e = bridge.New(nk, log, pool, dialers, opts)
		if e != nil {
			return
		}
		bridges = append(bridges, b)
	}
	var l listener.Listener
	for _, opts := range conf.Listener {
		l, e = listener.New(nk, log, pool, dialers, api, opts)
		if e != nil {
			return
		}
		listeners = append(listeners, l)
	}
	var u *udp.UDP
	for _, opts := range conf.UDP {
		u, e = udp.New(log, opts)
		if e != nil {
			return
		}
		udps = append(udps, u)
	}
	app.dialers = dialers
	app.bridges = bridges
	app.listeners = listeners
	app.udps = udps
	return
}
func (a *Application) Serve() {
	var wait sync.WaitGroup
	for _, item := range a.bridges {
		wait.Add(1)
		go serveWait(&wait, item)
	}
	for _, item := range a.listeners {
		wait.Add(1)
		go serveWait(&wait, item)
	}
	for _, udp := range a.udps {
		wait.Add(1)
		go serveWait(&wait, udp)
	}
	wait.Wait()
}

type iserve interface {
	Serve() error
}

func serveWait(wait *sync.WaitGroup, item iserve) {
	defer wait.Done()
	item.Serve()
}
