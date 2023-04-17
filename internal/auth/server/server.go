package server

import (
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
)

func Server() http.HandlerFunc {
	sign_in_route := conf.MustGetConfOrDefault("/sign-in", "auth", "signInPath")
	//sign_out_route := conf.MustGetConfOrDefault("/sign-out", "auth", "signOutPath")
	//log_in_route := conf.MustGetConfOrDefault("/log-in", "auth", "logInPath")
	//log_out_route := conf.MustGetConfOrDefault("/log-out", "auth", "logOutPath")

	r := mux.NewRouter()
	r.Route(sign_in_route, signInRoute)
	//r.Route(sign_out_route, routes.SignOutRoute(s, ch, createGRPCClient))

	return r.ServeHTTP
}

func Serve(w http.ResponseWriter, r *http.Request) {
	Server()(w, r)
}
