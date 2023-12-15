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
		listener = make([]Listener, 0, len(conf.Listener))
		l        Listener
	)
	for _, opts := range conf.Listener {
		switch opts.Mode {
		case "basic", "":
			l, e = NewBasicListener(log, &opts.BasicListener)
			if e != nil {
				for _, l = range listener {
					l.Close()
				}
				return
			}
			listener = append(listener, l)
		case "http":
			l, e = NewHttpListener(&opts.BasicListener)
			if e != nil {
				for _, l = range listener {
					l.Close()
				}
				return
			}
			listener = append(listener, l)
		default:
			e = fmt.Errorf(`listener mode: %s`, opts.Mode)
			log.Error(`listener mode`, `mode`, opts.Mode)
			return
		}
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
