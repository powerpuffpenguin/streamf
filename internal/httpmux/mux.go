package httpmux

import (
	"log/slog"
	"net/http"
)

type ServeMux struct {
	log  *slog.Logger
	mux  *http.ServeMux
	keys map[string]*Handler
}

func New(log *slog.Logger) *ServeMux {
	return &ServeMux{
		log:  log,
		mux:  http.NewServeMux(),
		keys: make(map[string]*Handler),
	}
}
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.mux.ServeHTTP(w, r)
}
func (mux *ServeMux) Get(pattern string, handler http.HandlerFunc) {
	found, ok := mux.keys[pattern]
	if ok {
		if found.get != nil {
			mux.log.Warn(`router is replaced`,
				`pattern`, pattern,
			)
		}
		found.get = handler
	} else {
		found = &Handler{
			get: handler,
		}
		mux.keys[pattern] = found
		mux.mux.Handle(pattern, found)
	}
}
func (mux *ServeMux) Head(pattern string, handler http.HandlerFunc) {
	found, ok := mux.keys[pattern]
	if ok {
		if found.head != nil {
			mux.log.Warn(`router is replaced`,
				`pattern`, pattern,
			)
		}
		found.head = handler
	} else {
		found = &Handler{
			head: handler,
		}
		mux.keys[pattern] = found
		mux.mux.Handle(pattern, found)
	}
}
func (mux *ServeMux) Post(pattern string, handler http.HandlerFunc) {
	found, ok := mux.keys[pattern]
	if ok {
		if found.post != nil {
			mux.log.Warn(`router is replaced`,
				`pattern`, pattern,
			)
		}
		found.post = handler
	} else {
		found = &Handler{
			post: handler,
		}
		mux.keys[pattern] = found
		mux.mux.Handle(pattern, found)
	}
}
func (mux *ServeMux) Put(pattern string, handler http.HandlerFunc) {
	found, ok := mux.keys[pattern]
	if ok {
		if found.put != nil {
			mux.log.Warn(`router is replaced`,
				`pattern`, pattern,
			)
		}
		found.put = handler
	} else {
		found = &Handler{
			put: handler,
		}
		mux.keys[pattern] = found
		mux.mux.Handle(pattern, found)
	}
}
func (mux *ServeMux) Patch(pattern string, handler http.HandlerFunc) {
	found, ok := mux.keys[pattern]
	if ok {
		if found.patch != nil {
			mux.log.Warn(`router is replaced`,
				`pattern`, pattern,
			)
		}
		found.patch = handler
	} else {
		found = &Handler{
			patch: handler,
		}
		mux.keys[pattern] = found
		mux.mux.Handle(pattern, found)
	}
}
func (mux *ServeMux) Delete(pattern string, handler http.HandlerFunc) {
	found, ok := mux.keys[pattern]
	if ok {
		if found.delete != nil {
			mux.log.Warn(`router is replaced`,
				`pattern`, pattern,
			)
		}
		found.delete = handler
	} else {
		found = &Handler{
			delete: handler,
		}
		mux.keys[pattern] = found
		mux.mux.Handle(pattern, found)
	}
}
