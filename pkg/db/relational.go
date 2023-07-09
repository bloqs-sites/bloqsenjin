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
}

type DataManipulater interface {
	Select(ctx context.Context, table string, columns func() map[string]any, where map[string]any) (Result, error)
	Insert(ctx context.Context, table string, rows []map[string]any) (Result, error)
	Update(ctx context.Context, table string, assignments map[string]any, conditions map[string]any) error
	Delete(ctx context.Context, table string, conditions map[string]any) error

	CreateTables(context.Context, []Table) error
	CreateIndexes(context.Context, []Index) error
	CreateViews(context.Context, []View) error
	DropTables(context.Context, []Table) error

	Close() error
}

type Result struct {
	LastID *int64
	Rows   []JSON
}

type JSON map[string]any
