package main

import (
	"bufio"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/internal/db"
	"github.com/bloqs-sites/bloqsenjin/internal/models"

	//"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

func init() {

}

func main() {
	conn := db.NewMariaDB("owduser:passwd@/owd")

	s := rest.NewServer(":8089", &conn)

	file, err := os.Open("./cmd/api/preferences")

	if err != nil {
		panic(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "=")

		if len(parts) == 1 {
			parts = append(parts, "")
		}

		go conn.Insert("preference", []map[string]string{
			{
				"name":        parts[0],
				"description": parts[1],
			},
		})
	}

	//auth := auth.NewAuthManager(nil)
	s.AttachHandler("preference", new(models.PreferenceHandler))
	s.AttachHandler("bloq", new(models.BloqHandler))
	//s.AttachHandler(
	//    "/preference",
	//    auth.AuthDecor(
	//        new(models.PreferenceHandler),
	//        1,
	//    )(),
	//)

	err = s.Run()
	if errors.Is(err, http.ErrServerClosed) {

	} else if err != nil {
		os.Exit(1)
	}
}
