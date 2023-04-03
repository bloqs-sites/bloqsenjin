package db

type Table struct {
	Name    string
	Columns []string
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
	Select(table string, columns func() map[string]any) (Result, error)
	Insert(table string, rows []map[string]string) (Result, error)
	Update(table string, assignments []map[string]any, conditions []map[string]any) (Result, error)
	Delete(table string, conditions []map[string]any) (Result, error)

	CreateTables([]Table) error
	CreateIndexes([]Index) error
	CreateViews([]View) error
}

type Result struct {
	LastID *int64
	Rows   []JSON
}

type JSON map[string]any
