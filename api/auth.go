package api

import (
	"net/http"

	auth "github.com/bloqs-sites/bloqsenjin/pkg/auth/http"
)

func Serve(w http.ResponseWriter, r *http.Request) {
	auth.Serve(w, r)
}
