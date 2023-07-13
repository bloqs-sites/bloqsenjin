package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/image"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	httpPort = flag.Int("HTTPPort", 8787, "The HTTP server port")
)

func main() {
	flag.Parse()
	if err := conf.Compile(); err != nil {
		switch err := err.(type) {
		case jsonschema.InvalidJSONTypeError:
			panic(err)
		}
	}

	fmt.Printf("`image` server port:\t %d\n", *httpPort)
	http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), image.Server("/"))
}
