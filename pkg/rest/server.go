package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	pb "github.com/bloqs-sites/bloqsenjin/proto"

	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
)

type Server struct {
	port string
	mux  *mux.Router
	dbh  *db.DataManipulater
	auth pb.AuthClient
}

func NewServer(port string, crud db.DataManipulater, auth pb.AuthClient) Server {
	return Server{
		port: port,
		mux:  mux.NewRouter(),
		dbh:  &crud,
		auth: auth,
	}
}

func (s Server) AttachHandler(ctx context.Context, route string, h Handler) {
	db := *s.dbh
	db.CreateTables(ctx, h.CreateTable())
	db.CreateIndexes(ctx, h.CreateIndexes())
	db.CreateViews(ctx, h.CreateViews())

	s.mux.Route(route, func(w http.ResponseWriter, r *http.Request) {
		models, err := h.Handle(r, s)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if len(models) == 0 {
			_, err = w.Write([]byte("{}"))
		} else if len(models) == 1 {
			err = json.NewEncoder(w).Encode(models[0])
		} else {
			err = json.NewEncoder(w).Encode(models)
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err.Error())
		}
	})
}

func (s *Server) GetDB() *db.DataManipulater {
	return s.dbh
}

func (s Server) Run() error {
	return http.ListenAndServe(s.port, s.mux)
}

func (s Server) ValidateJWT(r *http.Request, permitions uint64) bool {
	res, err := s.auth.Validate(r.Context(), &pb.Token{
		Jwt:         []byte(r.Header.Get("Authorization")),
		Permissions: &permitions,
	})

	if err != nil {
		return false
	}

	return res.Valid
}
