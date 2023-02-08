package rest

import (
	"net/http"
)

type JSON map[string]any

type CRUDer interface {
	Create(*http.Request, DataManipulater) ([]JSON, error)
	Read(*http.Request, DataManipulater) ([]JSON, error)
	Update(*http.Request, DataManipulater) ([]JSON, error)
	Delete(*http.Request, DataManipulater) ([]JSON, error)
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
	CRUDer
	Mapper
}
