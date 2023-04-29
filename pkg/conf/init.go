package conf

import (
	"flag"
	"os"
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
	sch_default_path = "https://bloqs.torres-dev.workers.dev/sch"
	sch_usage        = ""
	sch_env_var      = "BLOQS_SCHEMA"
)

func init() {
	path, exists := os.LookupEnv(cnf_env_var)
	if !exists {
		flag.StringVar(cnf_path, cnf_flag, cnf_default_path, cnf_usage)
	} else {
		flag.StringVar(cnf_path, cnf_flag, path, cnf_usage)
	}

	path, exists = os.LookupEnv(sch_env_var)
	if !exists {
		flag.StringVar(sch_path, sch_flag, sch_default_path, sch_usage)
	} else {
		flag.StringVar(sch_path, sch_flag, path, sch_usage)
	}
}
