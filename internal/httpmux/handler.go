package httpmux

import "net/http"

type Handler struct {
	get    http.HandlerFunc
	head   http.HandlerFunc
	post   http.HandlerFunc
	put    http.HandlerFunc
	patch  http.HandlerFunc
	delete http.HandlerFunc
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		if h.get != nil {
			h.get(w, req)
			return
		}
	case http.MethodHead:
		if h.head != nil {
			h.head(w, req)
			return
		} else if h.get != nil {
			h.get(w, req)
			return
		}
	case http.MethodPost:
		if h.post != nil {
			h.post(w, req)
			return
		}
	case http.MethodPut:
		if h.put != nil {
			h.put(w, req)
			return
		}
	case http.MethodPatch:
		if h.patch != nil {
			h.patch(w, req)
			return
		}
	case http.MethodDelete:
		if h.delete != nil {
			h.delete(w, req)
			return
		}
	}
	w.Header().Set(`Content-Type`, `text/plain; charset=utf-8`)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 page not found\n"))
}
