package conf

import (
	"flag"
	"os"

	_ "github.com/joho/godotenv/autoload"
)

var (
	cnf_path = flag.String(cnf_flag, cnf_default_path, cnf_usage)
	sch_path = flag.String(sch_flag, sch_default_path, sch_usage)
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
	if exists {
		*cnf_path = path
	}

	path, exists = os.LookupEnv(sch_env_var)
	if exists {
		*sch_path = path
	}
}
