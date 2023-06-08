package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	internal_db "github.com/bloqs-sites/bloqsenjin/internal/db"
	"github.com/bloqs-sites/bloqsenjin/internal/models"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

func Server(ctx context.Context) http.HandlerFunc {
	if err := conf.Compile(); err != nil {
		panic(err)
	}

	dbh, err := internal_db.NewMySQL(ctx, strings.TrimSpace(os.Getenv("BLOQS_REST_MYSQL_DSN")))
	if err != nil {
		panic(fmt.Errorf("error creating DB instance of type `%T`:\t%s", dbh, err))
	}

	s := rest.NewRESTServer(dbh)

	s.AttachHandler(context.Background(), "/preference", new(models.PreferenceHandler))
	s.AttachHandler(context.Background(), "/account", new(models.Account))
	//s.AttachHandler(context.Background(), "bloq", new(models.BloqHandler))

	return s.Serve()
}

func Serve(w http.ResponseWriter, r *http.Request) {
	Server(r.Context())(w, r)
}
