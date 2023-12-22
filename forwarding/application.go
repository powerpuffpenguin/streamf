package forwarding

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/powerpuffpenguin/streamf/bridge"
	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/listener"
	"github.com/powerpuffpenguin/streamf/pool"
)

type Application struct {
	bridges   []bridge.Bridge
	listeners []listener.Listener
	dialers   map[string]dialer.Dialer
	log       *slog.Logger
}

func NewApplication(conf *config.Config) (app *Application, e error) {
	log, e := newLogger(&conf.Logger)
	if e != nil {
		return
	}
	var (
		bridges   = make([]bridge.Bridge, 0, len(conf.Bridge))
		b         bridge.Bridge
		tag       string
		dialers   = make(map[string]dialer.Dialer, len(conf.Dialer))
		d         dialer.Dialer
		exists    bool
		listeners = make([]listener.Listener, 0, len(conf.Listener))
		l         listener.Listener
		pool      = pool.New(&conf.Pool)
		nk        = network.New()
	)
	app = &Application{
		log: log,
	}
	api := app.api()
	for _, opts := range conf.Dialer {
		tag = opts.Tag
		if _, exists = dialers[tag]; exists {
			e = fmt.Errorf(`dialer tag repeat: %s`, opts.Tag)
			log.Error(`dialer tag repeat`, `tag`, opts.Tag)
			return
		}
		d, e = dialer.New(nk, log, pool, opts)
		if e != nil {
			for _, d = range dialers {
				d.Close()
			}
			return
		}
		dialers[opts.Tag] = d
	}
	for _, opts := range conf.Bridge {
		b, e = bridge.New(nk, log, pool, dialers, opts)
		if e != nil {
			for _, d = range dialers {
				d.Close()
			}
			for _, b := range bridges {
				b.Close()
			}
			return
		}
		bridges = append(bridges, b)
	}
	for _, opts := range conf.Listener {
		l, e = listener.New(nk, log, pool, dialers, api, opts)
		if e != nil {
			for _, b := range bridges {
				b.Close()
			}
			for _, d = range dialers {
				d.Close()
			}
			for _, l = range listeners {
				l.Close()
			}
			return
		}
		listeners = append(listeners, l)
	}
	app.dialers = dialers
	app.bridges = bridges
	app.listeners = listeners
	return
}
func (a *Application) Serve() {
	nb := len(a.bridges)
	if nb == 0 {
		serve(a.listeners)
		return
	}
	nl := len(a.listeners)
	if nl == 0 {
		serve(a.bridges)
		return
	}

	var wait sync.WaitGroup
	wait.Add(nb + nl)
	for _, item := range a.bridges {
		go serveWait(&wait, item)
	}
	for _, item := range a.listeners {
		go serveWait(&wait, item)
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
func serve[T iserve](items []T) {
	n := len(items)
	switch n {
	case 0:
	case 1:
		items[0].Serve()
	case 2:
		done := make(chan struct{})
		go func() {
			defer close(done)
			items[0].Serve()
		}()
		items[1].Serve()
		<-done
	default:
		var wait sync.WaitGroup
		n--
		for i := 0; i < n; i++ {
			wait.Add(1)
			go serveWait(&wait, items[i])
		}
		items[n].Serve()
		wait.Wait()
	}
}
