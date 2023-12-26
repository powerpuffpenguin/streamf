package forwarding

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	"github.com/powerpuffpenguin/streamf/internal/httpmux"
	"github.com/powerpuffpenguin/streamf/version"
)

func (a *Application) api() []httpmux.ApiHandler {
	return []httpmux.ApiHandler{
		{
			Method:  []string{http.MethodGet},
			Path:    `/`,
			Handler: a.apiRoot,
		},
		{
			Method:  []string{http.MethodGet},
			Path:    `/application`,
			Handler: a.apiApplication,
		},
		{
			Method:  []string{http.MethodGet},
			Path:    `/listener`,
			Handler: a.apiListener,
		},
		{
			Method:  []string{http.MethodGet},
			Path:    `/dialer`,
			Handler: a.apiDialer,
		},
		{
			Method:  []string{http.MethodGet},
			Path:    `/bridge`,
			Handler: a.apiBridge,
		},
		{
			Method:  []string{http.MethodGet},
			Path:    `/runtime`,
			Handler: a.apiRuntime,
		},
	}
}
func (a *Application) apiApplication(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(`Content-Type`, `application/json; charset=utf-8`)
	m := make(map[string]any)
	items := make([]any, 0, len(a.listeners))
	for _, item := range a.listeners {
		items = append(items, item.Info())
	}
	m[`listeners`] = items

	items = make([]any, 0, len(a.dialers))
	for _, item := range a.dialers {
		items = append(items, item.Info())
	}
	m[`dialers`] = items

	items = make([]any, 0, len(a.bridges))
	for _, item := range a.bridges {
		items = append(items, item.Info())
	}
	m[`bridges`] = items

	jw := json.NewEncoder(w)
	if beauty := r.URL.Query().Get(`beauty`); beauty == `1` || beauty == `true` {
		jw.SetIndent("", "\t")
	}
	jw.Encode(m)
}
func (a *Application) apiListener(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(`Content-Type`, `application/json; charset=utf-8`)
	items := make([]any, 0, len(a.listeners))
	for _, item := range a.listeners {
		items = append(items, item.Info())
	}
	jw := json.NewEncoder(w)
	if beauty := r.URL.Query().Get(`beauty`); beauty == `1` || beauty == `true` {
		jw.SetIndent("", "\t")
	}
	jw.Encode(items)
}
func (a *Application) apiDialer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(`Content-Type`, `application/json; charset=utf-8`)
	items := make([]any, 0, len(a.dialers))
	for _, item := range a.dialers {
		items = append(items, item.Info())
	}
	jw := json.NewEncoder(w)
	if beauty := r.URL.Query().Get(`beauty`); beauty == `1` || beauty == `true` {
		jw.SetIndent("", "\t")
	}
	jw.Encode(items)
}
func (a *Application) apiBridge(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(`Content-Type`, `application/json; charset=utf-8`)
	items := make([]any, 0, len(a.bridges))
	for _, item := range a.bridges {
		items = append(items, item.Info())
	}
	jw := json.NewEncoder(w)
	if beauty := r.URL.Query().Get(`beauty`); beauty == `1` || beauty == `true` {
		jw.SetIndent("", "\t")
	}
	jw.Encode(items)
}
func (a *Application) apiRuntime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(`Content-Type`, `application/json; charset=utf-8`)
	jw := json.NewEncoder(w)
	if beauty := r.URL.Query().Get(`beauty`); beauty == `1` || beauty == `true` {
		jw.SetIndent("", "\t")
	}
	jw.Encode(map[string]any{
		`platform`: fmt.Sprintf(`%s/%s, %s, %s, %s`,
			runtime.GOOS, runtime.GOARCH,
			runtime.Version(),
			version.Date, version.Commit,
		),
		`version`:   version.Version,
		`goroutine`: runtime.NumGoroutine(),
		`cgo`:       runtime.NumCgoCall(),
	})
}

func (a *Application) apiRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(`Content-Type`, `text/html; charset=utf-8`)
	w.Write([]byte(`<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>streamf</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
<p>
	<h1><a href="https://github.com/powerpuffpenguin/streamf">github</a></h1>
	This is a port forwarding program written in golang. 
	But it doesn't just forward port data, it also supports forwarding streams. 
	That is, data of a certain stream protocol is forwarded to another stream.
</p>
<p>
	<h1>API</h1>
	<ul>
	<li><a href="application?beauty=1">application</a></li>
	<li><a href="listener?beauty=1">listener</a></li>
	<li><a href="dialer?beauty=1">dialer</a></li>
	<li><a href="bridge?beauty=1">bridge</a></li>
	<li><a href="runtime?beauty=1">runtime</a></li>
	</ul>
</p>
</body></html>`))
}
