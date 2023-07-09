package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	db "github.com/bloqs-sites/bloqsenjin/internal/db"
	"github.com/bloqs-sites/bloqsenjin/internal/models"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

func Server(ctx context.Context, endpoint string) http.HandlerFunc {
	if err := conf.Compile(); err != nil {
		panic(err)
	}

	dbh, err := db.NewMySQL(ctx, strings.TrimSpace(os.Getenv("BLOQS_REST_MYSQL_DSN")))
	if err != nil {
		panic(fmt.Errorf("error creating DB instance of type `%T`:\t%s", dbh, err))
	}

	s := rest.NewRESTServer(endpoint, dbh)

	s.AttachHandler(context.Background(), "/preference", new(models.Preference))
	s.AttachHandler(context.Background(), "/profile", new(models.Profile))
	s.AttachHandler(context.Background(), "/bloq", new(models.Bloq))
	s.AttachHandler(context.Background(), "/offer", new(models.Offer))

	return s.Serve()
}

func Serve(endpoint string, w http.ResponseWriter, r *http.Request) {
	Server(r.Context(), endpoint)(w, r)
}
