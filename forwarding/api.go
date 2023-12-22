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
	w.Header().Set(`Content-Type`, `text/plain; charset=utf-8`)
	w.Write([]byte(`/apiApplication`))
}
func (a *Application) apiListener(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`/apiListener`))
}
func (a *Application) apiDialer(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`/apiDialer`))
}
func (a *Application) apiBridge(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(`Content-Type`, `application/json; charset=utf-8`)
	bridges := make([]any, 0, len(a.bridges))
	for _, item := range a.bridges {
		bridges = append(bridges, item.Info())
	}
	json.NewEncoder(w).Encode(bridges)
}
func (a *Application) apiRuntime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(`Content-Type`, `application/json; charset=utf-8`)
	json.NewEncoder(w).Encode(map[string]any{
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
	<li><a href="application">application</a></li>
	<li><a href="listener">listener</a></li>
	<li><a href="dialer">dialer</a></li>
	<li><a href="bridge">bridge</a></li>
	<li><a href="runtime">runtime</a></li>
	</ul>
</p>
</body></html>`))
}
