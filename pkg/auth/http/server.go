package http

import (
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
)

func Server() http.HandlerFunc {
	sign_route := conf.MustGetConfOrDefault("/sign", "auth", "signPath")
	//log_route := conf.MustGetConfOrDefault("/log", "auth", "logPath")

	r := mux.NewRouter()
	r.Route(sign_route, signRoute)

	return r.ServeHTTP
}

func Serve(w http.ResponseWriter, r *http.Request) {
	Server()(w, r)
}
