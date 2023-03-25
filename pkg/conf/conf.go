package conf

import (
	"encoding/json"
	"flag"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
    _ "github.com/santhosh-tekuri/jsonschema/v5/httploader"
)

var (
	conf_path = flag.String(conf_flag, conf_default_path, conf_usage)
    compiler = jsonschema.NewCompiler()
    sch *jsonschema.Schema

    conf map[string]any
)

const (
    conf_flag = "bloqs-conf"
    conf_default_path = "./.bloqs.conf.json"
    conf_usage = ""

    schema_path = "https://black-silence-a2dc.torres-dev.workers.dev/"
)

func init() {
    flag.Parse()

    compiler.Draft = jsonschema.Draft2020

    var err error
    if sch, err = compiler.Compile(schema_path); err != nil {
        panic(err)
    }

    if conf, err = readConf(*conf_path); err != nil {
        panic(err)
    }

    if err = sch.Validate(conf); err != nil {
        panic(err)
    }
}

func GetConf() map[string]any {
    return conf
}

func readConf(path string) (map[string]any, error) {
    var r io.ReadCloser

    if _, err := url.Parse(path); err != nil {
        path, err := filepath.Abs(path)
        if err != nil {
            return nil, err
        }

        file, err := os.Open(path)
        if err != nil {
            return nil, err
        }

        defer file.Close()

        r = io.ReadCloser(file)
    } else {
        r, err = jsonschema.LoadURL(path)

        if err != nil {
            return nil, err
        }
    }

    defer r.Close()

    var buf map[string]any
    if err := json.NewDecoder(r).Decode(&buf); err != nil {
        return buf, err
    }

    return buf, nil
}
