package db

import "context"

type Table struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

type Index struct {
	Name  string
	Table string
	Cols  []string
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

type DataManipulater interface {
	Select(ctx context.Context, table string, columns func() map[string]any) (Result, error)
	Insert(ctx context.Context, table string, rows []map[string]string) (Result, error)
	Update(ctx context.Context, table string, assignments []map[string]any, conditions []map[string]any) (Result, error)
	Delete(ctx context.Context, table string, conditions []map[string]any) (Result, error)

	CreateTables(context.Context, []Table) error
	CreateIndexes(context.Context, []Index) error
	CreateViews(context.Context, []View) error
}

type Result struct {
	LastID *int64
	Rows   []JSON
}

type JSON map[string]any
