package conf

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
	_ "github.com/santhosh-tekuri/jsonschema/v5/httploader"
	"google.golang.org/protobuf/internal/errors"
)

var (
	conf_path = flag.String(conf_flag, conf_default_path, conf_usage)
	compiler  = jsonschema.NewCompiler()
	sch       *jsonschema.Schema

	conf map[string]any
)

const (
	conf_flag         = "bloqs-conf"
	conf_default_path = "./.bloqs.conf.json"
	conf_usage        = ""

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

func GetConf(keys ...string) (any, error) {
	c := conf
	for _, i := range keys {
		v, ok := c[i]
		if !ok {
			return nil, errors.New("nil")
		}

		if m, ok := v.(map[string]any); ok {
			c = m
		} else {
			return v, nil
		}
	}

	return c, nil
}

func MustGetConf(keys ...string) any {
	if v, err := GetConf(keys...); err == nil {
		return v
	}

	return nil
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
