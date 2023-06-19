package rest

import (
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/pkg/db"
)

type Created struct {
	LastID  *int64 `json:"id"`
	Status  uint16 `json:"status"`
	Message string `json:"message"`
}

type Resource struct {
	Models  []db.JSON `json:"models"`
	Status  uint16    `json:"status"`
	Message string    `json:"message"`
	Unique  bool      `json:"unique"`
}

type CRUDer interface {
	Create(http.ResponseWriter, *http.Request, RESTServer) (*Created, error)
	Read(http.ResponseWriter, *http.Request, RESTServer) (*Resource, error)
	Update(http.ResponseWriter, *http.Request, RESTServer) (*Resource, error)
	Delete(http.ResponseWriter, *http.Request, RESTServer) (*Resource, error)
}

type Handler interface {
	CRUDer
	db.Mapper
	Table() string
}
