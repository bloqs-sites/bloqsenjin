package http

import (
	"encoding/json"
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
)

func Server() http.HandlerFunc {
	sign_route := conf.MustGetConfOrDefault("/sign", "auth", "signPath")
	log_route := conf.MustGetConfOrDefault("/log", "auth", "logPath")
	types_route := conf.MustGetConfOrDefault("/types", "auth", "typesPath")

	r := mux.NewRouter()
	r.Route(sign_route, signRoute)
	r.Route(log_route, logRoute)
	r.Route(types_route, func(w http.ResponseWriter, r *http.Request) {
		// XXX
		json.NewEncoder(w).Encode(conf.MustGetConfOrDefault([]any{}, "auth", "supported"))
	})

	return r.ServeHTTP
}

func Serve(w http.ResponseWriter, r *http.Request) {
	Server()(w, r)
}
