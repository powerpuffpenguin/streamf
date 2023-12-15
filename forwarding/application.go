package forwarding

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/powerpuffpenguin/sf/config"
)

type Application struct {
	listener []Listener
	log      *slog.Logger
}

func NewApplication(conf *config.Config) (app *Application, e error) {
	log, e := newLogger(&conf.Logger)
	if e != nil {
		return
	}
	var (
		tag      string
		dialer   = make(map[string]Dialer, len(conf.Dialer))
		d        Dialer
		exists   bool
		listener = make([]Listener, 0, len(conf.Listener))
		l        Listener
	)
	for _, opts := range conf.Dialer {
		tag = opts.Tag
		if _, exists = dialer[tag]; exists {
			e = fmt.Errorf(`dialer tag repeat: %s`, opts.Tag)
			log.Error(`dialer tag repeat`, `tag`, opts.Tag)
			return
		}
		d, e = NewDialer(log, opts)
		if e != nil {
			for _, d = range dialer {
				d.Close()
			}
			return
		}
		dialer[opts.Tag] = d
	}
	for _, opts := range conf.Listener {
		l, e = NewListener(log, dialer, opts)
		if e != nil {
			for _, l = range listener {
				l.Close()
			}
			return
		}
		listener = append(listener, l)
	}
	app = &Application{
		listener: listener,
		log:      log,
	}
	return
}
func (a *Application) Serve() {
	listener := a.listener
	n := len(listener)
	switch n {
	case 0:
	case 1:
		listener[0].Serve()
	case 2:
		listener[0].Serve()
		done := make(chan struct{})
		go func() {
			defer close(done)
			listener[1].Serve()
		}()
		<-done
	default:
		var wait sync.WaitGroup
		n--
		for i := 0; i < n; i++ {
			wait.Add(1)
			go func(l Listener) {
				defer wait.Done()
				l.Serve()
			}(listener[i])
		}
		listener[n].Serve()
		wait.Wait()
	}
}
