package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	internal_auth "github.com/bloqs-sites/bloqsenjin/internal/auth"
	"github.com/bloqs-sites/bloqsenjin/internal/db"
	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/proto"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	email    = flag.String("email", "", "")
	password = flag.String("password", "", "")
)

func init() {
	flag.Parse()

	if email == nil || *email == "" {
		panic("no email specified")
	}

	if password == nil || *password == "" {
		panic("no email specified")
	}

	if err := conf.Compile(); err != nil {
		switch err := err.(type) {
		case jsonschema.InvalidJSONTypeError:
			panic(err)
		}
	}

}

func main() {
	ctx := context.Background()
	a, err := authSrv(ctx)
	if err != nil {
		panic(err)
	}

	user := &proto.Credentials_Basic{
		Basic: &proto.Credentials_BasicCredentials{
			Email:    *email,
			Password: *password,
		},
	}
	creds := &proto.Credentials{Credentials: user}

	if err = a.SignInBasic(ctx, user); err != nil {
		panic(err)
	}

	if err = a.GrantSuper(ctx, creds); err != nil {
		a.SignOutBasic(ctx, user)
		panic(err)
	}
}

func authSrv(ctx context.Context) (auth.Auther, error) {
	creds, err := db.NewMySQL(ctx, strings.TrimSpace(os.Getenv("BLOQS_AUTH_MYSQL_DSN")))
	if err != nil {
		return nil, fmt.Errorf("error creating DB instance of type `%T`:\t%s", creds, err)
	}

	return internal_auth.NewBloqsAuther(ctx, creds)
}
