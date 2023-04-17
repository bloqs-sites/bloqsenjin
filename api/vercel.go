package api

import "github.com/bloqs-sites/bloqsenjin/internal/auth/server"

func Serve(w http.ResponseWriter, r *http.Request) {
	server.Serve(w, r)
}
