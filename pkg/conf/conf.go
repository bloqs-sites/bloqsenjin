package conf

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
	_ "github.com/santhosh-tekuri/jsonschema/v5/httploader"
)

var (
	c   = jsonschema.NewCompiler()
	sch *jsonschema.Schema

	cnf map[string]any
)

func init() {
	c.Draft = jsonschema.Draft2020
}

func Compile() error {
	var err error
	if sch, err = c.Compile(*SchPath); err != nil {
		return err
	}

	if cnf, err = readConf(*CnfPath); err != nil {
		return err
	}

	return sch.Validate(map[string]any(cnf))
}

func GetConf(keys ...string) (any, error) {
	if cnf == nil {
		panic(errors.New("needs to conf.Compile"))
		//return nil, errors.New("needs to conf.Compile");
	}

	c := cnf
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

func MustGetConfOrDefault[T any](default_value T, keys ...string) T {
	if v, err := GetConf(keys...); err == nil {
		return v.(T)
	}

	return default_value
}

func readConf(path string) (map[string]any, error) {
	var r io.ReadCloser

	if _, err := url.ParseRequestURI(path); err != nil {
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
