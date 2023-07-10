package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/internal/db"
	"github.com/bloqs-sites/bloqsenjin/internal/models"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

func main() {
	if err := conf.Compile(); err != nil {
		panic(err)
	}

	dbh, err := db.NewMySQL(context.Background(), strings.TrimSpace(os.Getenv("BLOQS_REST_MYSQL_DSN")))
	if err != nil {
		panic(fmt.Errorf("error creating DB instance of type `%T`:\t%s", dbh, err))
	}

	for _, i := range []rest.Handler{
		new(models.Profile),
		new(models.Preference),
		new(models.Bloq),
		new(models.Org),
		new(models.Offer),
		new(models.Order),
	} {
		if err := dbh.DropTables(context.Background(), i.CreateTable()); err != nil {
			panic(err)
		}
	}
}
