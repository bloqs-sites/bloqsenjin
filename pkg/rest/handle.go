package rest

import (
	"net/http"
)

type JSON map[string]any

type CRUDer interface {
	Create(*http.Request, Server) ([]JSON, error)
	Read(*http.Request, Server) ([]JSON, error)
	Update(*http.Request, Server) ([]JSON, error)
	Delete(*http.Request, Server) ([]JSON, error)
}

type Table struct {
	Name    string
	Columns []string
}

type Mapper interface {
	CreateTable() []Table
	MapGenerator() func() map[string]any
}

type Handler interface {
	Handle(*http.Request, Server) ([]JSON, error)

	CRUDer
	Mapper
}
