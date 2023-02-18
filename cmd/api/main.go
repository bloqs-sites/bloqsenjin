package main

import (
	"errors"
	"net/http"
	"os"

	"github.com/bloqs-sites/bloqsenjin/internal/db"
	"github.com/bloqs-sites/bloqsenjin/internal/models"
	//"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

func main() {
	conn := db.NewMariaDB("owduser:passwd@/owd")

	s := rest.NewServer(":8080", &conn)

	//auth := auth.NewAuthManager(nil)
	s.AttachHandler(
		"/preference",
		new(models.PreferenceHandler),
	)
	s.AttachHandler(
		"/bloq",
		new(models.BloqHandler),
	)
	//s.AttachHandler(
	//    "/preference",
	//    auth.AuthDecor(
	//        new(models.PreferenceHandler),
	//        1,
	//    )(),
	//)

	err := s.Run()
	if errors.Is(err, http.ErrServerClosed) {

	} else if err != nil {
		os.Exit(1)
	}
}
