package rest

import (
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/pkg/db"
)

type CRUDer interface {
	Create(*http.Request, Server) ([]db.JSON, error)
	Read(*http.Request, Server) ([]db.JSON, error)
	Update(*http.Request, Server) ([]db.JSON, error)
	Delete(*http.Request, Server) ([]db.JSON, error)
}

type Handler interface {
	Handle(*http.Request, Server) ([]db.JSON, error)

	CRUDer
	db.Mapper
}
