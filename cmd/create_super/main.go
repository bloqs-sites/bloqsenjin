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
	"github.com/redis/go-redis/v9"
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
	srv, err := authSrv(ctx)
	if err != nil {
		panic(err)
	}

	v, err := srv.SignIn(ctx, &proto.Credentials{
		Credentials: &proto.Credentials_Basic{
			Basic: &proto.Credentials_BasicCredentials{
				Email:    *email,
				Password: *password,
			},
		},
	})

	if err != nil {
		panic(err)
	}

	if !v.Valid {
		panic(v.Message)
	}
}

func authSrv(ctx context.Context) (proto.AuthServer, error) {
	// TODO: How can I make it that you can specify which implementation of the interfaces you want to use?
	creds, err := db.NewMySQL(ctx, strings.TrimSpace(os.Getenv("BLOQS_AUTH_MYSQL_DSN")))
	if err != nil {
		return nil, fmt.Errorf("error creating DB instance of type `%T`:\t%s", creds, err)
	}

	opt, err := redis.ParseURL(strings.TrimSpace(os.Getenv("BLOQS_TOKENS_REDIS_DSN")))
	if err != nil {
		return nil, fmt.Errorf("could not parse the `BLOQS_TOKENS_REDIS_DSN` to create the credentials to connect to the DB:\t%s", err)
	}

	a, err := internal_auth.NewBloqsAuther(ctx, creds)
	if err != nil {
		return nil, err
	}

	secrets, err := db.NewKeyDB(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error creating DB instance of type `%T`:\t%s", secrets, err)
	}

	t := internal_auth.NewBloqsTokener(secrets)

	return auth.NewAuthServer(a, t), nil
}
