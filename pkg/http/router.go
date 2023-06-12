package http

import (
	"net/http"
	"strings"
)

type Handle func(w http.ResponseWriter, r *http.Request, segs []string)
type Router struct {
	endpoint string
	routes   map[string]Handle
}

func NewRouter(endpoint string) *Router {
	endpoint = strings.TrimRightFunc(endpoint, func(r rune) bool {
		return r == '/'
	})

	return &Router{
		endpoint: endpoint,
		routes:   make(map[string]Handle),
	}
}

func (mux *Router) Route(route string, h Handle) {
	parts := strings.Split(route, "/")
	route = parts[1]
	mux.routes[route] = h
}

func (mux *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, mux.endpoint) {
		http.NotFound(w, r)
		return
	}

	path := r.URL.Path[len(mux.endpoint):]

	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		http.NotFound(w, r)
		return
	}

	var route string
	if len(parts) == 1 {
		route = ""
	} else {
		route = parts[1]
	}

	handler, ok := mux.routes[route]
	if !ok {
		http.NotFound(w, r)
		return
	}

	handler(w, r, parts[2:])
}
