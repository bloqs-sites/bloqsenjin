package http

import (
	"encoding/json"
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
)

func Server(endpoint string) http.HandlerFunc {
	if err := conf.Compile(); err != nil {
		panic(err)

		// switch err := err.(type) {
		// case jsonschema.InvalidJSONTypeError:
		// 	panic(err)
		// }
	}

	sign_route := conf.MustGetConfOrDefault("/sign/", "auth", "paths", "sign")
	log_route := conf.MustGetConfOrDefault("/log/", "auth", "paths", "log")
	types_route := conf.MustGetConfOrDefault("/types/", "auth", "paths", "types")

	r := mux.NewRouter(endpoint)
	r.Route(sign_route, SignRoute)
	r.Route(log_route, LogRoute)
	r.Route(types_route, func(w http.ResponseWriter, r *http.Request, segs []string) {
		// XXX
		json.NewEncoder(w).Encode(conf.MustGetConfOrDefault(map[string]bool{}, "auth", "supported"))
	})

	return r.ServeHTTP
}

func Serve(endpoint string, w http.ResponseWriter, r *http.Request) {
	Server(endpoint)(w, r)
}
