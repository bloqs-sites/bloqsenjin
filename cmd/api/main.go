package main

import (
	"errors"
	"net/http"
	"os"

	"github.com/bloqs-sites/bloqsenjin/internal/db"
	"github.com/bloqs-sites/bloqsenjin/internal/models"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

func main() {
	conn := db.NewMariaDB("owduser:passwd@/owd")

	s := rest.NewServer(":8080", &conn)

	s.AttachHandler("/preference", new(models.PreferenceHandler))

	err := s.Run()
	if errors.Is(err, http.ErrServerClosed) {

	} else if err != nil {
		os.Exit(1)
	}
}
