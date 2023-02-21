package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Router struct {
    routes map[string]func(w http.ResponseWriter, r *http.Request)
}

type Server struct {
	port string
	mux  *Router
	dbh  *DataManipulater
}

func NewServer(port string, crud DataManipulater) Server {
	return Server{
		port: port,
		mux:  &Router{
            routes: make(map[string]func(w http.ResponseWriter, r *http.Request)),
        },
		dbh:  &crud,
	}
}

type Result struct {
	LastID *int64
	Rows   []JSON
}

type DataManipulater interface {
	Select(table string, columns func() map[string]any) (Result, error)
	Insert(table string, rows []map[string]string) (Result, error)
	Update(table string, assignments []map[string]any, conditions []map[string]any) (Result, error)
	Delete(table string, conditions []map[string]any) (Result, error)

	CreateTables([]Table) error
}

func (s Server) AttachHandler(route string, h Handler) {
	db := *s.dbh
	db.CreateTables(h.CreateTable())

	s.mux.routes[route] = func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func (s *Server) GetDB() *DataManipulater {
	return s.dbh
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    parts := strings.Split(r.URL.Path, "/")
    if len(parts) == 0 {
        http.NotFound(w, r)
        return
    }
    route := parts[1]

    handler, ok := s.mux.routes[route]
    if !ok {
        http.NotFound(w, r)
        return
    }

    handler(w, r)
}

func (s Server) Run() error {
	return http.ListenAndServe(s.port, s)
}
