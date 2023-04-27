package server

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	cnf_path *string
	sch_path *string
)

const (
	cnf_flag         = "bloqs-conf"
	cnf_default_path = "./.bloqs.conf.json"
	cnf_usage        = ""
	cnf_env_var      = "BLOQS_CONF"

	sch_flag         = "bloqs-schema"
	sch_default_path = "https://black-silence-a2dc.torres-dev.workers.dev/"
	sch_usage        = ""
	sch_env_var      = "BLOQS_SCHEMA"
)

func init() {
    path := os.Getenv(cnf_env_var)
    cnf_path = &path
    flag.StringVar(cnf_path, cnf_flag, cnf_default_path, cnf_usage)

    path = os.Getenv(sch_env_var)
    sch_path = &path
    flag.StringVar(sch_path, sch_flag, sch_default_path, sch_usage)
}

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
	if err := conf.Compile(*sch_path, *cnf_path); err != nil {
		var err_msg strings.Builder
		switch err := err.(type) {
		case *jsonschema.SchemaError:
			err_msg.WriteString(fmt.Sprintf("failed to compile schema `%s`.\n", err.SchemaURL))
			switch err := err.Err.(type) {
			case *jsonschema.ValidationError:
				err_msg.WriteString(fmt.Sprintf("schema is not valid: %s.\n", err.DetailedOutput().Error))
			default:
				err_msg.WriteString("unknown error.\n")
			}
		case *jsonschema.ValidationError:
			err_msg.WriteString(fmt.Sprintf("schema is not valid: %s.\n", err.Error()))
		case *jsonschema.InfiniteLoopError:
			err_msg.WriteString(fmt.Sprintf("schema compilation/validation found infinite loop at `%s`.\n", err.Error()))
		case *jsonschema.InvalidJSONTypeError:
			err_msg.WriteString(fmt.Sprintf("received invalid JSON: %s.\n", err.Error()))
		case *os.PathError:
			err_msg.WriteString(fmt.Sprintf("error in operation `%s` on file `%s`: %s.\n.\n", err.Op, err.Path, err.Error()))
		default:
			err_msg.WriteString(fmt.Sprintf("unexpected error: %s.\n", err.Error()))
		}

		w.Write([]byte(err_msg.String()))
		w.Header().Add("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	Server()(w, r)
}
