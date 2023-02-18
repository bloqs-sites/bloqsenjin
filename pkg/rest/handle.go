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

type Index struct {
	Name   string
	Table   string
	Cols   []string
}

type View struct {
	Name   string
	Select string
}

type Mapper interface {
	CreateTable() []Table
	CreateIndexes() []Index
	CreateViews() []View
	MapGenerator() func() map[string]any
}

type Handler interface {
	Handle(*http.Request, Server) ([]JSON, error)

	CRUDer
	Mapper
}
