package http

import (
	"net/http"
	"strings"
)

type Handle func(w http.ResponseWriter, r *http.Request)
type Router struct {
	routes map[string]Handle
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]Handle),
	}
}

func (mux *Router) Route(route string, h Handle) {
	parts := strings.Split(route, "/")
	route = parts[1]
	mux.routes[route] = h
}

func (mux *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) == 0 {
		http.NotFound(w, r)
		return
	}

	route := parts[1]

	handler, ok := mux.routes[route]
	if !ok {
		http.NotFound(w, r)
		return
	}

	handler(w, r)
}
