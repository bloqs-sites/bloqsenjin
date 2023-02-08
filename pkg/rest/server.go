package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Server struct {
	port string
	mux  *http.ServeMux
	dbh  *DataManipulater
}

func NewServer(port string, crud DataManipulater) Server {
	return Server{
		port: port,
		mux:  http.NewServeMux(),
		dbh:  &crud,
	}
}

type Result struct {
	LastID *int64
	Rows   []JSON
}

type DataManipulater interface {
	Select(table string, columns func() map[string]any) (Result, error)
	Insert(table string, rows []map[string]any) (Result, error)
	Update(table string, assignments []map[string]any, conditions []map[string]any) (Result, error)
	Delete(table string, conditions []map[string]any) (Result, error)

    CreateTables([]Table) error
}

func (s *Server) AttachHandler(route string, h Handler) {
    db := *s.dbh
    db.CreateTables(h.CreateTable())

	s.mux.Handle(route, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		models, err := s.Guide(r, h)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(models)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err.Error())
		}
	}))
}

func (s Server) Run() error {
	return http.ListenAndServe(s.port, s.mux)
}

func (s Server) Guide(r *http.Request, h Handler) ([]JSON, error) {
	switch r.Method {
	case "":
		fallthrough
	case http.MethodGet:
		return h.Read(r, *s.dbh)
	case http.MethodPost:
		return h.Create(r, *s.dbh)
	}

	return nil, errors.New(fmt.Sprint(http.StatusMethodNotAllowed))
}
