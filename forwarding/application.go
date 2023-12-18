package forwarding

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/powerpuffpenguin/sf/config"
	"github.com/powerpuffpenguin/sf/dialer"
	"github.com/powerpuffpenguin/sf/internal/network"
	"github.com/powerpuffpenguin/sf/listener"
	"github.com/powerpuffpenguin/sf/pool"
)

type Application struct {
	listeners []listener.Listener
	log       *slog.Logger
}

func NewApplication(conf *config.Config) (app *Application, e error) {
	log, e := newLogger(&conf.Logger)
	if e != nil {
		return
	}
	var (
		tag       string
		dialers   = make(map[string]dialer.Dialer, len(conf.Dialer))
		d         dialer.Dialer
		exists    bool
		listeners = make([]listener.Listener, 0, len(conf.Listener))
		l         listener.Listener
		pool      = pool.New(&conf.Pool)
		nk        = network.New()
	)
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
	for _, opts := range conf.Listener {
		l, e = listener.New(nk, log, pool, dialers, opts)
		if e != nil {
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
	app = &Application{
		listeners: listeners,
		log:       log,
	}
	return
}
func (a *Application) Serve() {
	listeners := a.listeners
	n := len(listeners)
	switch n {
	case 0:
	case 1:
		listeners[0].Serve()
	case 2:
		done := make(chan struct{})
		go func() {
			defer close(done)
			listeners[0].Serve()
		}()
		listeners[1].Serve()
		<-done
	default:
		var wait sync.WaitGroup
		n--
		for i := 0; i < n; i++ {
			wait.Add(1)
			go func(l listener.Listener) {
				defer wait.Done()
				l.Serve()
			}(listeners[i])
		}
		listeners[n].Serve()
		wait.Wait()
	}
}
