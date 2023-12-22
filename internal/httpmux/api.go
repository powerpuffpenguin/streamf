package httpmux

import "net/http"

type ApiHandler struct {
	Method  []string
	Path    string
	Handler http.HandlerFunc
}
