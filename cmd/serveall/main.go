package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	auth "github.com/bloqs-sites/bloqsenjin/pkg/auth/http"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/fortune"
	rest "github.com/bloqs-sites/bloqsenjin/pkg/rest/http"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	authPort    = flag.Int("authPort", 3000, "The HTTP auth server port")
	restPort    = flag.Int("restPort", 8080, "The HTTP rest server port")
	fortunePort = flag.Int("fortunePort", 4747, "The HTTP fortune server port")
)

func main() {
	flag.Parse()
	if err := conf.Compile(); err != nil {
		switch err := err.(type) {
		case jsonschema.InvalidJSONTypeError:
			panic(err)
		}
	}

	ch := make(chan error)

	go func() {
		fmt.Printf("`auth` server port:\t %d\n", *authPort)
		ch <- http.ListenAndServe(fmt.Sprintf(":%d", *authPort), auth.Server("/"))
	}()
	go func() {
		fmt.Printf("`rest` server port:\t %d\n", *restPort)
		ch <- http.ListenAndServe(fmt.Sprintf(":%d", *restPort), rest.Server(context.Background(), "/"))
	}()
	go func() {
		fmt.Printf("`fortune` server port:\t %d\n", *fortunePort)
		ch <- http.ListenAndServe(fmt.Sprintf(":%d", *fortunePort), fortune.Server("/"))
	}()

	for i := range ch {
		if i != nil {
			panic(i)
		}
	}
}
